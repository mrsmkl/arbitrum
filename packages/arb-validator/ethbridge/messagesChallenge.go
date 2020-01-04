/*
 * Copyright 2020, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ethbridge

import (
	"context"
	"math/big"
	"strings"

	"github.com/offchainlabs/arbitrum/packages/arb-validator/ethbridge/messageschallenge"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	errors2 "github.com/pkg/errors"
)

var messagesBisectedID common.Hash
var messagesOneStepProofCompletedID common.Hash

func init() {
	parsed, err := abi.JSON(strings.NewReader(messageschallenge.MessagesChallengeABI))
	if err != nil {
		panic(err)
	}
	messagesBisectedID = parsed.Events["Bisected"].ID()
	messagesOneStepProofCompletedID = parsed.Events["OneStepProofCompleted"].ID()
}

type MessagesChallenge struct {
	*BisectionChallenge
	Challenge *messageschallenge.MessagesChallenge
}

func NewMessagesChallenge(address common.Address, client *ethclient.Client) (*MessagesChallenge, error) {
	bisectionChallenge, err := NewBisectionChallenge(address, client)
	if err != nil {
		return nil, err
	}
	vm := &MessagesChallenge{BisectionChallenge: bisectionChallenge}
	err = vm.setupContracts()
	return vm, err
}

func (c *MessagesChallenge) setupContracts() error {
	challengeManagerContract, err := messageschallenge.NewMessagesChallenge(c.address, c.Client)
	if err != nil {
		return errors2.Wrap(err, "Failed to connect to MessagesChallenge")
	}

	c.Challenge = challengeManagerContract
	return nil
}

func (c *MessagesChallenge) StartConnection(ctx context.Context, outChan chan Notification, errChan chan error) error {
	if err := c.BisectionChallenge.StartConnection(ctx, outChan, errChan); err != nil {
		return err
	}
	if err := c.setupContracts(); err != nil {
		return err
	}
	header, err := c.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	filter := ethereum.FilterQuery{
		Addresses: []common.Address{c.address},
		Topics: [][]common.Hash{{
			messagesBisectedID,
			messagesOneStepProofCompletedID,
		}},
	}

	logs, err := c.Client.FilterLogs(ctx, filter)
	if err != nil {
		return err
	}
	for _, log := range logs {
		if err := c.processEvents(ctx, log, outChan); err != nil {
			return err
		}
	}

	filter.FromBlock = header.Number
	logChan := make(chan types.Log)
	logSub, err := c.Client.SubscribeFilterLogs(ctx, filter, logChan)
	if err != nil {
		return err
	}

	go func() {
		defer logSub.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				break
			case log := <-logChan:
				if err := c.processEvents(ctx, log, outChan); err != nil {
					errChan <- err
					return
				}
			case err := <-logSub.Err():
				errChan <- err
				return
			}
		}
	}()
	return nil
}

func (c *MessagesChallenge) processEvents(ctx context.Context, log types.Log, outChan chan Notification) error {
	event, err := func() (Event, error) {
		if log.Topics[0] == messagesBisectedID {
			eventVal, err := c.Challenge.ParseBisected(log)
			if err != nil {
				return nil, err
			}
			return MessagesBisectionEvent{
				ChainHashes:   eventVal.ChainHashes,
				SegmentHashes: eventVal.SegmentHashes,
				TotalLength:   eventVal.TotalLength,
				DeadlineTicks: eventVal.DeadlineTicks,
			}, nil
		} else if log.Topics[0] == messagesOneStepProofCompletedID {
			_, err := c.Challenge.ParseOneStepProofCompleted(log)
			if err != nil {
				return nil, err
			}
			return OneStepProofEvent{}, nil
		}
		return nil, errors2.New("unknown arbitrum event type")
	}()

	if err != nil {
		return err
	}

	header, err := c.Client.HeaderByHash(ctx, log.BlockHash)
	if err != nil {
		return err
	}
	outChan <- Notification{
		Header: header,
		VMID:   c.address,
		Event:  event,
		TxHash: log.TxHash,
	}
	return nil
}

func (c *MessagesChallenge) Bisect(
	auth *bind.TransactOpts,
	chainHashes [][32]byte,
	segmentHashes [][32]byte,
	chainLength *big.Int,
) (*types.Receipt, error) {

	tx, err := c.Challenge.Bisect(
		auth,
		chainHashes,
		segmentHashes,
		chainLength,
	)
	if err != nil {
		return nil, err
	}
	return waitForReceipt(auth.Context, c.Client, auth.From, tx, "Bisect")
}

func (c *MessagesChallenge) OneStepProof(
	auth *bind.TransactOpts,
	lowerHashA [32]byte,
	topHashA [32]byte,
	lowerHashB [32]byte,
	topHashB [32]byte,
	value [32]byte,
) (*types.Receipt, error) {
	tx, err := c.Challenge.OneStepProof(
		auth,
		lowerHashA,
		topHashA,
		lowerHashB,
		topHashB,
		value,
	)
	if err != nil {
		return nil, err
	}
	return waitForReceipt(auth.Context, c.Client, auth.From, tx, "OneStepProof")
}