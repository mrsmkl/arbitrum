package challenge

import (
	"context"
	"fmt"
	"math/big"

	"github.com/offchainlabs/arbitrum/packages/arb-node-core/ethbridge"
	"github.com/offchainlabs/arbitrum/packages/arb-util/core"
	"github.com/offchainlabs/arbitrum/packages/arb-util/machine"
	"github.com/pkg/errors"
)

type ExecutionImpl struct {
}

func (e *ExecutionImpl) SegmentTarget() int {
	return 400
}

var unreachableCut core.SimpleCut = core.NewSimpleCut([32]byte{})

func getCut(execTracker *core.ExecutionTracker, maxTotalMessagesRead *big.Int, gasTarget *big.Int) (core.Cut, *big.Int, error) {
	state, steps, err := execTracker.GetExecutionState(gasTarget)
	// mach, err := execTracker.GetMachine(gasTarget)
	// mach_hash, err := mach.Hash()
	// fmt.Printf("got cut %v gas target %v machine %v\n", state, gasTarget, mach_hash)
	if err != nil {
		return nil, nil, err
	}
	if state.TotalMessagesRead.Cmp(maxTotalMessagesRead) > 0 || state.TotalGasConsumed.Cmp(gasTarget) < 0 {
		// Execution read more messages than provided so assertion should have
		// stopped short
		return unreachableCut, steps, nil
	}
	return state, steps, nil
}

func (e *ExecutionImpl) GetCuts(lookup core.ArbCoreLookup, assertion *core.Assertion, offsets []*big.Int) ([]core.Cut, error) {
	execTracker := core.NewExecutionTracker(lookup, true, offsets, true)
	cuts := make([]core.Cut, 0, len(offsets))
	for i, offset := range offsets {
		cut, _, err := getCut(execTracker, assertion.After.TotalMessagesRead, offset)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			_, ok := cut.(*core.ExecutionState)
			if !ok {
				return nil, errors.New("first cut is unreachable")
			}
		}

		cuts = append(cuts, cut)
	}
	return cuts, nil
}

type DivergenceInfo struct {
	DifferentIndex   int
	SegmentSteps     *big.Int
	EndIsUnreachable bool
}

func (e *ExecutionImpl) FindFirstDivergence(lookup core.ArbCoreLookup, assertion *core.Assertion, offsets []*big.Int, cuts []core.Cut) (DivergenceInfo, error) {
	errRes := DivergenceInfo{
		DifferentIndex:   0,
		SegmentSteps:     big.NewInt(0),
		EndIsUnreachable: false,
	}
	fmt.Printf("search divergence %v cuts %v\n", offsets, cuts)
	execTracker := core.NewExecutionTracker(lookup, true, offsets, true)
	lastSteps := big.NewInt(0)
	for i, offset := range offsets {
		localCut, newSteps, err := getCut(execTracker, assertion.After.TotalMessagesRead, offset)
		if err != nil {
			return errRes, err
		}
		if localCut.CutHash() != cuts[i].CutHash() {
			fmt.Printf("found divergent cut at %v from %v: local %v other %v hash %v other hash %v\n", offset, offsets, localCut, cuts[i], localCut.CutHash(), cuts[i].CutHash())
			return DivergenceInfo{
				DifferentIndex:   i,
				SegmentSteps:     new(big.Int).Sub(newSteps, lastSteps),
				EndIsUnreachable: localCut == unreachableCut,
			}, nil
		}
		lastSteps = newSteps
	}
	fmt.Printf("no divergence %v cuts %v\n", offsets, cuts)
	return errRes, errors.New("no divergence found in cuts")
}

func (e *ExecutionImpl) Bisect(
	ctx context.Context,
	challenge *ethbridge.Challenge,
	prevBisection *core.Bisection,
	segmentToChallenge int,
	inconsistentSegment *core.ChallengeSegment,
	subCuts []core.Cut,
) error {
	return challenge.BisectExecution(
		ctx,
		prevBisection,
		segmentToChallenge,
		inconsistentSegment,
		subCuts,
	)
}

func (e *ExecutionImpl) getSegmentStartInfo(lookup core.ArbCoreLookup, assertion *core.Assertion, segment *core.ChallengeSegment) (*core.ExecutionState, machine.Machine, error) {
	execTracker := core.NewExecutionTracker(lookup, true, []*big.Int{segment.Start}, true)
	cut, _, err := getCut(execTracker, assertion.After.TotalMessagesRead, segment.Start)
	if err != nil {
		return nil, nil, err
	}
	execCut, ok := cut.(*core.ExecutionState)
	if !ok {
		return nil, nil, errors.New("attempted to one step prove blocked machine")
	}

	beforeMachine, err := execTracker.GetMachine(segment.Start)
	if err != nil {
		return nil, nil, err
	}

	return execCut, beforeMachine, nil
}

func (e *ExecutionImpl) OneStepProof(
	ctx context.Context,
	challenge *ethbridge.Challenge,
	lookup core.ArbCoreLookup,
	assertion *core.Assertion,
	prevBisection *core.Bisection,
	segmentToChallenge int,
	challengedSegment *core.ChallengeSegment,
) (byte, machine.Machine, error) {
	previousCut, previousMachine, err := e.getSegmentStartInfo(lookup, assertion, challengedSegment)
	if err != nil {
		return 0, nil, err
	}

	proofData, bufferProofData, err := previousMachine.MarshalForProof()
	if err != nil {
		return 0, nil, err
	}

	opcode := proofData[0]

	fmt.Printf("buffer %v, op %v\n", bufferProofData, opcode)

	return opcode, previousMachine, challenge.OneStepProveExecution(
		ctx,
		prevBisection,
		segmentToChallenge,
		challengedSegment,
		previousCut,
		proofData,
		bufferProofData,
		opcode,
	)
}

func (e *ExecutionImpl) OneStepProofMachine(
	ctx context.Context,
	challenge *ethbridge.Challenge,
	lookup core.ArbCoreLookup,
	assertion *core.Assertion,
	challengedSegment *core.ChallengeSegment,
) (byte, machine.Machine, error) {
	_, previousMachine, err := e.getSegmentStartInfo(lookup, assertion, challengedSegment)
	if err != nil {
		return 0, nil, err
	}

	proofData, bufferProofData, err := previousMachine.MarshalForProof()
	if err != nil {
		return 0, nil, err
	}

	opcode := proofData[0]

	fmt.Printf("buffer %v, op %v\n", bufferProofData, opcode)

	return opcode, previousMachine, nil
}

func (e *ExecutionImpl) OneStepProofInfo(
	ctx context.Context,
	challenge *ethbridge.Challenge,
	lookup core.ArbCoreLookup,
	assertion *core.Assertion,
	prevBisection *core.Bisection,
	segmentToChallenge int,
	challengedSegment *core.ChallengeSegment,
) (byte, machine.Machine, error) {
	_, previousMachine, err := e.getSegmentStartInfo(lookup, assertion, challengedSegment)
	if err != nil {
		return 0, nil, err
	}

	proofData, bufferProofData, err := previousMachine.MarshalForProof()
	if err != nil {
		return 0, nil, err
	}

	opcode := proofData[0]

	fmt.Printf("buffer %v, op %v\n", bufferProofData, opcode)

	return opcode, previousMachine, nil
}

func (e *ExecutionImpl) ProveContinuedExecution(
	ctx context.Context,
	challenge *ethbridge.Challenge,
	lookup core.ArbCoreLookup,
	assertion *core.Assertion,
	prevBisection *core.Bisection,
	segmentToChallenge int,
	challengedSegment *core.ChallengeSegment,
) error {
	previousCut, _, err := e.getSegmentStartInfo(lookup, assertion, challengedSegment)
	if err != nil {
		return err
	}

	return challenge.ProveContinuedExecution(
		ctx,
		prevBisection,
		segmentToChallenge,
		challengedSegment,
		previousCut,
	)
}
