/*
 * Copyright 2019, Offchain Labs, Inc.
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

package ethvalidator

import (
	"bytes"
	"context"
	"fmt"
	solsha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/offchainlabs/arb-validator/valmessage"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/arb-avm/protocol"
	"github.com/offchainlabs/arb-avm/value"

	"github.com/offchainlabs/arb-validator/challengeRPC"
	"github.com/offchainlabs/arb-validator/verifierRPC"
)

type EthConnection struct {
	client         *ethclient.Client
	Tracker        *verifierRPC.VMTracker
	Challenge      *challengeRPC.ChallengeManager
	OneStep        *challengeRPC.OneStepProof
	BalanceTracker *verifierRPC.ArbBalanceTracker
}

type ArbAddresses struct {
	TrackerAddress        string `json:"vmTracker"`
	ChallengeAddress      string `json:"ChallengeManager"`
	OneStepAddress        string `json:"OneStepProof"`
	BalanceTrackerAddress string `json:"balanceTracker"`
}

func NewEthConnection(serverAddress string, a ArbAddresses) (*EthConnection, error) {
	client, err := ethclient.Dial(serverAddress)
	if err != nil {
		return nil, err
	}

	trackerContract, err := verifierRPC.NewVMTracker(common.HexToAddress(a.TrackerAddress), client)
	if err != nil {
		return nil, err
	}
	challangeManagerContract, err := challengeRPC.NewChallengeManager(common.HexToAddress(a.ChallengeAddress), client)
	if err != nil {
		return nil, err
	}
	onestepProofContract, err := challengeRPC.NewOneStepProof(common.HexToAddress(a.OneStepAddress), client)
	if err != nil {
		return nil, err
	}

	balanceTrackerContract, err := verifierRPC.NewArbBalanceTracker(common.HexToAddress(a.BalanceTrackerAddress), client)
	if err != nil {
		return nil, err
	}

	return &EthConnection{client, trackerContract, challangeManagerContract, onestepProofContract, balanceTrackerContract}, nil
}

func (con *EthConnection) CreateListeners(vmId [32]byte) (chan interface{}, chan error, error) {
	headers := make(chan *types.Header)
	headersSub, err := con.client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		return nil, nil, err
	}

	vmCreatedChan := make(chan *verifierRPC.VMTrackerVMCreated)
	start := uint64(0)
	watch := &bind.WatchOpts{
		Context: context.Background(),
		Start:   &start,
	}

	vmCreatedSub, err := con.Tracker.WatchVMCreated(watch, vmCreatedChan)
	if err != nil {
		return nil, nil, err
	}

	messageDeliveredChan := make(chan *verifierRPC.VMTrackerMessageDelivered)
	messageDeliveredSub, err := con.Tracker.WatchMessageDelivered(watch, messageDeliveredChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	unanAssChan := make(chan *verifierRPC.VMTrackerFinalUnanimousAssertion)
	unanAssSub, err := con.Tracker.WatchFinalUnanimousAssertion(watch, unanAssChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	unanPropChan := make(chan *verifierRPC.VMTrackerProposedUnanimousAssertion)
	unanPropSub, err := con.Tracker.WatchProposedUnanimousAssertion(watch, unanPropChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	unanConfChan := make(chan *verifierRPC.VMTrackerConfirmedUnanimousAssertion)
	unanConfSub, err := con.Tracker.WatchConfirmedUnanimousAssertion(watch, unanConfChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	dispAssChan := make(chan *verifierRPC.VMTrackerDisputableAssertion)
	dispAssSub, err := con.Tracker.WatchDisputableAssertion(watch, dispAssChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	confAssChan := make(chan *verifierRPC.VMTrackerConfirmedAssertion)
	confAssSub, err := con.Tracker.WatchConfirmedAssertion(watch, confAssChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	challengeInitiatedChan := make(chan *verifierRPC.VMTrackerInitiatedChallenge)
	challengeInitiatedSub, err := con.Tracker.WatchInitiatedChallenge(watch, challengeInitiatedChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	challengeBisectedChan := make(chan *challengeRPC.ChallengeManagerBisectedAssertion)
	challengeBisectedSub, err := con.Challenge.WatchBisectedAssertion(watch, challengeBisectedChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	challengeContinuedChan := make(chan *challengeRPC.ChallengeManagerContinuedChallenge)
	challengeContinuedSub, err := con.Challenge.WatchContinuedChallenge(watch, challengeContinuedChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	challengeTimedOutChan := make(chan *challengeRPC.ChallengeManagerTimedOutChallenge)
	challengeTimedOutSub, err := con.Challenge.WatchTimedOutChallenge(watch, challengeTimedOutChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	oneStepProofChan := make(chan *challengeRPC.ChallengeManagerOneStepProofCompleted)
	oneStepProofSub, err := con.Challenge.WatchOneStepProofCompleted(watch, oneStepProofChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	//dispAssDebugChan := make(chan *verifierRPC.VMTrackerDisputableAssertionDebug)
	//dispAssDebugSub, err := con.Tracker.WatchDisputableAssertionDebug(watch, dispAssDebugChan, [][32]byte{VmId})
	//if err != nil {
	//	return nil, nil, err
	//}

	//unanAssDebugChan := make(chan *verifierRPC.VMTrackerUnanimousAssertionDebug)
	//unanAssDebugSub, err := con.Tracker.WatchUnanimousAssertionDebug(watch, unanAssDebugChan, [][32]byte{VmId})
	//if err != nil {
	//	return nil, nil, err
	//}

	challengeOneStepDebugChan := make(chan *challengeRPC.ChallengeManagerOneStepProofDebug)
	challengeOneStepDebugSub, err := con.Challenge.WatchOneStepProofDebug(watch, challengeOneStepDebugChan, [][32]byte{vmId})
	if err != nil {
		return nil, nil, err
	}

	outChan := make(chan interface{}, 1024)
	errChan := make(chan error, 1024)
	go func() {
		defer close(outChan)
		defer close(errChan)
		defer headersSub.Unsubscribe()
		defer messageDeliveredSub.Unsubscribe()
		defer messageDeliveredSub.Unsubscribe()
		defer vmCreatedSub.Unsubscribe()
		defer unanAssSub.Unsubscribe()
		defer dispAssSub.Unsubscribe()
		defer confAssSub.Unsubscribe()
		defer challengeInitiatedSub.Unsubscribe()
		defer challengeBisectedSub.Unsubscribe()
		defer challengeInitiatedSub.Unsubscribe()
		defer challengeContinuedSub.Unsubscribe()
		defer oneStepProofSub.Unsubscribe()
		//defer dispAssDebugSub.Unsubscribe()
		//defer unanAssDebugSub.Unsubscribe()
		defer challengeOneStepDebugSub.Unsubscribe()
		err := func () error {
			for {
				select {
				case header := <-headers:
					outChan <- header
				case val := <-messageDeliveredChan:
					outChan <- val
				case val := <-vmCreatedChan:
					outChan <- val
				case val := <-unanAssChan:
					outChan <- val
				case val := <-unanPropChan:
					outChan <- val
				case val := <-unanConfChan:
					outChan <- val
				case val := <-dispAssChan:
					outChan <- val
				case val := <-confAssChan:
					outChan <- val
				case val := <-challengeInitiatedChan:
					outChan <- val
				case val := <-challengeBisectedChan:
					outChan <- val
				case val := <-challengeTimedOutChan:
					outChan <- val
				case val := <-challengeContinuedChan:
					outChan <- val
				case val := <-oneStepProofChan:
					outChan <- val
				//case Val := <-dispAssDebugChan:
				//	outChan <- Val
				//case Val := <- unanAssDebugChan:
				//	outChan <- Val
				case val := <-challengeOneStepDebugChan:
					outChan <- val
				case err := <-headersSub.Err():
					return err
				case err := <-messageDeliveredSub.Err():
					return err
				case err := <-vmCreatedSub.Err():
					return err
				case err := <-unanAssSub.Err():
					return err
				case err := <-unanPropSub.Err():
					return err
				case err := <-unanConfSub.Err():
					return err
				case err := <-dispAssSub.Err():
					return err
				case err := <-confAssSub.Err():
					return err
				case err := <-challengeInitiatedSub.Err():
					return err
				case err := <-challengeBisectedSub.Err():
					return err
				case err := <-challengeContinuedSub.Err():
					return err
				case err := <-challengeTimedOutSub.Err():
					return err
				case err := <-oneStepProofSub.Err():
					return err
				case err := <-challengeOneStepDebugSub.Err():
					return err
				}
			}
		}()
		if err != nil {
			errChan <- err
		}
	}()
	return outChan, errChan, nil
}

func (con *EthConnection) SendMessage(
	auth *bind.TransactOpts,
	msg protocol.Message,
) (*types.Transaction, error) {
	var dataBuf bytes.Buffer
	if err := value.MarshalValue(msg.Data, &dataBuf); err != nil {
		return nil, err
	}
	fmt.Println("Sending valmessage to VM")
	return con.Tracker.SendMessage(
		auth,
		msg.Destination,
		msg.TokenType,
		msg.Currency,
		dataBuf.Bytes(),
	)
}

func (con *EthConnection) SendEthMessage(
	auth *bind.TransactOpts,
	data value.Value,
	destination [32]byte,
	amount *big.Int,
) (*types.Transaction, error) {
	var dataBuf bytes.Buffer
	if err := value.MarshalValue(data, &dataBuf); err != nil {
		return nil, err
	}
	return con.Tracker.SendEthMessage(
		&bind.TransactOpts{
			From:     auth.From,
			Signer:   auth.Signer,
			GasLimit: auth.GasLimit,
			Value:    amount,
		},
		destination,
		dataBuf.Bytes(),
	)
}

func (con *EthConnection) CreateVM(
	auth *bind.TransactOpts,
	data *CreateVMValidatorRequest,
	messageHash [32]byte,
	signatures []valmessage.Signature,
) (*types.Transaction, error) {
	sigData := make([]byte, 0, len(signatures)*65)
	for _, sig := range signatures {
		sigData = append(sigData, sig.R[:]...)
		sigData = append(sigData, sig.S[:]...)
		sigData = append(sigData, sig.V)
	}
	var owner common.Address
	copy(owner[:], data.Config.Owner.Value)
	var escrowCurrency common.Address
	copy(escrowCurrency[:], data.Config.EscrowCurrency.Value)
	return con.Tracker.CreateVm(
		auth,
		[3][32]byte{
			value.NewHashFromBuf(data.VmId),
			value.NewHashFromBuf(data.VmState),
			messageHash,
		},
		uint32(data.Config.GracePeriod),
		data.Config.MaxExecutionStepCount,
		uint16(data.ChallengeManagerNum),
		value.NewBigIntFromBuf(data.Config.EscrowRequired),
		escrowCurrency,
		owner,
		sigData,
	)
}

func (con *EthConnection) DepositFunds(auth *bind.TransactOpts, amount *big.Int, dest [32]byte) (*types.Transaction, error) {
	return con.BalanceTracker.DepositEth(
		&bind.TransactOpts{
			From:     auth.From,
			Signer:   auth.Signer,
			GasLimit: auth.GasLimit,
			Value:    amount,
		},
		dest,
	)
}

func (con *EthConnection) UnanimousAssert(
	auth *bind.TransactOpts,
	vmId [32]byte,
	newInboxHash [32]byte,
	timeBounds protocol.TimeBounds,
	assertion *protocol.Assertion,
	signatures []valmessage.Signature,
) (*types.Transaction, error) {
	sigData := make([]byte, 0, len(signatures)*65)
	for _, sig := range signatures {
		sigData = append(sigData, sig.R[:]...)
		sigData = append(sigData, sig.S[:]...)
		sigData = append(sigData, sig.V)
	}

	tokenNums := make([]uint16, 0, len(assertion.OutMsgs))
	amounts := make([]*big.Int, 0, len(assertion.OutMsgs))
	destinations := make([][32]byte, 0, len(assertion.OutMsgs))
	var messageData bytes.Buffer
	balance := protocol.NewBalanceTrackerFromMessages(assertion.OutMsgs)
	for _, msg := range assertion.OutMsgs {
		tokenNums = append(tokenNums, uint16(balance.TokenIndex(msg.TokenType, msg.Currency)))
		amounts = append(amounts, msg.Currency)
		destinations = append(destinations, msg.Destination)
		err := msg.Data.Marshal(&messageData)
		if err != nil {
			return nil, err
		}
	}

	return con.Tracker.UnanimousAssert(
		auth,
		vmId,
		assertion.AfterHash,
		newInboxHash,
		assertion.LogsHash(),
		timeBounds,
		balance.TokenTypes,
		messageData.Bytes(),
		tokenNums,
		amounts,
		destinations,
		sigData,
	)
}

func (con *EthConnection) ProposeUnanimousAssert(
	auth *bind.TransactOpts,
	vmId [32]byte,
	newInboxHash [32]byte,
	timeBounds protocol.TimeBounds,
	assertion *protocol.Assertion,
	sequenceNum uint64,
	signatures []valmessage.Signature,
) (*types.Transaction, error) {
	sigData := make([]byte, 0, len(signatures)*65)
	for _, sig := range signatures {
		sigData = append(sigData, sig.R[:]...)
		sigData = append(sigData, sig.S[:]...)
		sigData = append(sigData, sig.V)
	}

	balance := protocol.NewBalanceTrackerFromMessages(assertion.OutMsgs)
	tokenNums := make([]uint16, 0, len(assertion.OutMsgs))
	amounts := make([]*big.Int, 0, len(assertion.OutMsgs))
	destinations := make([][32]byte, 0, len(assertion.OutMsgs))
	var messageData bytes.Buffer
	for _, msg := range assertion.OutMsgs {
		tokenNums = append(tokenNums, uint16(balance.TokenIndex(msg.TokenType, msg.Currency)))
		amounts = append(amounts, msg.Currency)
		destinations = append(destinations, msg.Destination)
		err := msg.Data.Marshal(&messageData)
		if err != nil {
			return nil, err
		}
	}

	var unanRest [32]byte
	copy(unanRest[:], solsha3.SoliditySHA3(
		solsha3.Bytes32(newInboxHash),
		solsha3.Bytes32(assertion.AfterHash),
		messageData.Bytes(),
		bytes32ArrayEncoded(destinations),
	))

	return con.Tracker.ProposeUnanimousAssert(
		auth,
		vmId,
		unanRest,
		sequenceNum,
		timeBounds,
		balance.TokenTypes,
		tokenNums,
		amounts,
		sigData,
	)
}

func (con *EthConnection) ConfirmUnanimousAsserted(
	auth *bind.TransactOpts,
	vmId [32]byte,
	newInboxHash [32]byte,
	assertion *protocol.Assertion,
) (*types.Transaction, error) {
	var messageData bytes.Buffer
	tokenNums := make([]uint16, 0, len(assertion.OutMsgs))
	amounts := make([]*big.Int, 0, len(assertion.OutMsgs))
	destinations := make([][32]byte, 0, len(assertion.OutMsgs))
	balance := protocol.NewBalanceTrackerFromMessages(assertion.OutMsgs)
	for _, msg := range assertion.OutMsgs {
		tokenNums = append(tokenNums, uint16(balance.TokenIndex(msg.TokenType, msg.Currency)))
		amounts = append(amounts, msg.Currency)
		destinations = append(destinations, msg.Destination)
		err := msg.Data.Marshal(&messageData)
		if err != nil {
			return nil, err
		}
	}

	tx, err := con.Tracker.ConfirmUnanimousAsserted(
		auth,
		vmId,
		assertion.AfterHash,
		assertion.LogsHash(),
		newInboxHash,
		balance.TokenTypes,
		messageData.Bytes(),
		tokenNums,
		amounts,
		destinations,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't confirm assertion: %v", err)
	}
	return tx, nil
}

func (con *EthConnection) DisputableAssert(
	auth *bind.TransactOpts,
	vmId [32]byte,
	precondition *protocol.Precondition,
	assertion *protocol.Assertion,
) (*types.Transaction, error) {
	tokenNums := make([]uint16, 0, len(assertion.OutMsgs))
	amounts := make([]*big.Int, 0, len(assertion.OutMsgs))
	destinations := make([][32]byte, 0, len(assertion.OutMsgs))
	dataHashes := make([][32]byte, 0, len(assertion.OutMsgs))
	for _, msg := range assertion.OutMsgs {
		dataHashes = append(dataHashes, msg.Data.Hash())
		tokenNums = append(tokenNums, uint16(precondition.BeforeBalance.TokenIndex(msg.TokenType, msg.Currency)))
		amounts = append(amounts, msg.Currency)
		destinations = append(destinations, msg.Destination)
	}
	return con.Tracker.DisputableAssert(
		auth,
		[5][32]byte{
			vmId,
			precondition.BeforeHash,
			precondition.BeforeInbox.Hash(),
			assertion.AfterHash,
			assertion.LogsHash(),
		},
		assertion.NumSteps,
		precondition.TimeBounds,
		precondition.BeforeBalance.TokenTypes,
		dataHashes,
		tokenNums,
		amounts,
		destinations,
	)
}

func (con *EthConnection) ConfirmAsserted(
	auth *bind.TransactOpts,
	vmId [32]byte,
	precondition *protocol.Precondition,
	assertion *protocol.Assertion,
) (*types.Transaction, error) {
	var messageData bytes.Buffer
	tokenNums := make([]uint16, 0, len(assertion.OutMsgs))
	amounts := make([]*big.Int, 0, len(assertion.OutMsgs))
	destinations := make([][32]byte, 0, len(assertion.OutMsgs))
	for _, msg := range assertion.OutMsgs {
		tokenNums = append(tokenNums, uint16(precondition.BeforeBalance.TokenIndex(msg.TokenType, msg.Currency)))
		amounts = append(amounts, msg.Currency)
		destinations = append(destinations, msg.Destination)
		err := msg.Data.Marshal(&messageData)
		if err != nil {
			return nil, err
		}
	}

	tx, err := con.Tracker.ConfirmAsserted(
		auth,
		vmId,
		precondition.Hash(),
		assertion.AfterHash,
		assertion.LogsHash(),
		assertion.NumSteps,
		precondition.BeforeBalance.TokenTypes,
		messageData.Bytes(),
		tokenNums,
		amounts,
		destinations,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't confirm assertion: %v", err)
	}
	return tx, nil
}

func (con *EthConnection) InitiateChallenge(
	auth *bind.TransactOpts,
	vmId [32]byte,
	precondition *protocol.Precondition,
	assertion *protocol.AssertionStub,
) (*types.Transaction, error) {
	var preAssHash [32]byte
	copy(preAssHash[:], solsha3.SoliditySHA3(
		solsha3.Bytes32(precondition.Hash()),
		solsha3.Bytes32(assertion.Hash()),
	))
	return con.Tracker.InitiateChallenge(
		auth,
		vmId,
		preAssHash,
	)
}

func (con *EthConnection) BisectChallenge(
	auth *bind.TransactOpts,
	vmId [32]byte,
	deadline uint64,
	precondition *protocol.Precondition,
	assertions []*protocol.Assertion,
) (*types.Transaction, error) {
	afterHashAndMessageAndLogsBisections := make([][32]byte, 0, len(assertions)*3+2)
	totalMessageAmounts := make([]*big.Int, 0)
	totalSteps := uint32(0)
	stubs := make([]*protocol.AssertionStub, 0, len(assertions))
	for _, assertion := range assertions {
		stubs = append(stubs, assertion.Stub())
	}
	for _, assertion := range stubs {
		afterHashAndMessageAndLogsBisections = append(afterHashAndMessageAndLogsBisections, assertion.AfterHash)
		totalMessageAmounts = append(totalMessageAmounts, assertion.TotalVals...)
		totalSteps += assertion.NumSteps
	}
	afterHashAndMessageAndLogsBisections = append(afterHashAndMessageAndLogsBisections, stubs[0].FirstMessageHash)
	for _, assertion := range stubs {
		afterHashAndMessageAndLogsBisections = append(afterHashAndMessageAndLogsBisections, assertion.LastMessageHash)
	}
	afterHashAndMessageAndLogsBisections = append(afterHashAndMessageAndLogsBisections, stubs[0].FirstLogHash)
	for _, assertion := range stubs {
		afterHashAndMessageAndLogsBisections = append(afterHashAndMessageAndLogsBisections, assertion.LastLogHash)
	}
	return con.Challenge.BisectAssertion(
		auth,
		[3][32]byte{
			vmId,
			precondition.BeforeHash,
			precondition.BeforeInbox.Hash(),
		},
		afterHashAndMessageAndLogsBisections,
		totalMessageAmounts,
		totalSteps,
		precondition.TimeBounds,
		precondition.BeforeBalance.TokenTypes,
		precondition.BeforeBalance.TokenAmounts,
		deadline,
	)
}

func (con *EthConnection) ContinueChallenge(
	auth *bind.TransactOpts,
	vmId [32]byte,
	assertionToChallenge *big.Int,
	bisectionProof []byte,
	bisectionRoot [32]byte,
	bisectionHash [32]byte,
	deadline uint64,
) (*types.Transaction, error) {
	return con.Challenge.ContinueChallenge(
		auth,
		vmId,
		assertionToChallenge,
		bisectionProof,
		deadline,
		bisectionRoot,
		bisectionHash,
	)
}

func (con *EthConnection) OneStepProof(
	auth *bind.TransactOpts,
	vmId [32]byte,
	precondition *protocol.Precondition,
	assertion *protocol.AssertionStub,
	proof []byte,
	deadline uint64,
) (*types.Transaction, error) {
	return con.Challenge.OneStepProof(
		auth,
		vmId,
		[2][32]byte{precondition.BeforeHash, precondition.BeforeInbox.Hash()},
		precondition.TimeBounds,
		precondition.BeforeBalance.TokenTypes,
		precondition.BeforeBalance.TokenAmounts,
		[5][32]byte{
			assertion.AfterHash,
			assertion.FirstMessageHash,
			assertion.LastMessageHash,
			assertion.FirstLogHash,
			assertion.LastLogHash,
		},
		assertion.TotalVals,
		proof,
		deadline,
	)
}

func (con *EthConnection) AsserterTimedOutChallenge(
	auth *bind.TransactOpts,
	vmId [32]byte,
	bisectionHash [32]byte,
	deadline uint64,
) (*types.Transaction, error) {
	return con.Challenge.AsserterTimedOut(
		auth,
		vmId,
		bisectionHash,
		deadline,
	)
}

func TranslateBisectionEvent(event *challengeRPC.ChallengeManagerBisectedAssertion) []*protocol.AssertionStub {
	bisectionCount := (len(event.AfterHashAndMessageAndLogsBisections) - 2) / 3
	assertions := make([]*protocol.AssertionStub, 0, bisectionCount)
	stepCount := event.TotalSteps / uint32(bisectionCount)
	tokenTypeCount := len(event.TotalMessageAmounts) / bisectionCount
	for i := 0; i < bisectionCount; i++ {
		steps := stepCount
		if uint32(i) < event.TotalSteps%uint32(bisectionCount) {
			steps++
		}
		assertion := &protocol.AssertionStub{
			AfterHash:        event.AfterHashAndMessageAndLogsBisections[i],
			NumSteps:         steps,
			FirstMessageHash: event.AfterHashAndMessageAndLogsBisections[bisectionCount+i],
			LastMessageHash:  event.AfterHashAndMessageAndLogsBisections[bisectionCount+i+1],
			FirstLogHash:     event.AfterHashAndMessageAndLogsBisections[bisectionCount*2+1+i],
			LastLogHash:      event.AfterHashAndMessageAndLogsBisections[bisectionCount*2+2+1],
			TotalVals:        event.TotalMessageAmounts[i*tokenTypeCount : (i+1)*tokenTypeCount],
		}
		assertions = append(assertions, assertion)
	}
	return assertions
}

func TranslateDisputableAssertionEvent(event *verifierRPC.VMTrackerDisputableAssertion) (*protocol.Precondition, *protocol.AssertionStub) {
	balanceTracker := protocol.NewBalanceTrackerFromLists(event.TokenTypes, event.Amounts)
	precondition := protocol.NewPrecondition(
		event.Fields[0],
		event.TimeBounds,
		balanceTracker,
		value.NewHashOnlyValue(event.Fields[1], 1),
	)
	assertion := &protocol.AssertionStub{AfterHash: event.Fields[2], NumSteps: event.NumSteps, LastMessageHash: event.LastMessageHash, TotalVals: event.Amounts}
	return precondition, assertion
}
