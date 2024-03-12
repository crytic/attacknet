package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

type clientType string

const (
	Execution clientType = "execution"
	Consensus clientType = "consensus"
	Validator clientType = "validator"
)

func ConvertToNodeIdTag(networkNodeCount int, node *network.Node, client clientType) string {
	nodeNumStr := ""

	if networkNodeCount < 10 {
		nodeNumStr = fmt.Sprintf("%d", node.Index)
	} else if networkNodeCount < 100 {
		nodeNumStr = fmt.Sprintf("%02d", node.Index)
	} else {
		nodeNumStr = fmt.Sprintf("%03d", node.Index)
	}

	switch client {
	case Execution:
		return fmt.Sprintf("el-%s-%s-%s", nodeNumStr, node.Execution.Type, node.Consensus.Type)
	case Consensus:
		return fmt.Sprintf("cl-%s-%s-%s", nodeNumStr, node.Consensus.Type, node.Execution.Type)
	case Validator:
		return fmt.Sprintf("cl-%s-%s-%s-validator", nodeNumStr, node.Consensus.Type, node.Execution.Type)
	default:
		log.Errorf("Unrecognized node type %s", client)
		return ""
	}
}

func composeWaitForFaultCompletionStep() *types.PlanStep {
	return &types.PlanStep{StepType: types.WaitForFaultCompletion, StepDescription: "wait for faults to terminate"}
}

func composeNodeClockSkewPlanSteps(targetsSelected []*ChaosTargetSelector, skew, duration string) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	for _, target := range targetsSelected {
		description := fmt.Sprintf("Inject clock skew on target %s", target.Description)

		skewStep, err := buildClockSkewFault(description, skew, duration, target.Selector)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *skewStep)
	}

	return steps, nil
}

func composeNodeRestartSteps(targetsSelected []*ChaosTargetSelector) ([]types.PlanStep, error) {
	var steps []types.PlanStep

	for _, target := range targetsSelected {
		description := fmt.Sprintf("Restart target %s", target.Description)
		restartStep, err := buildPodRestartFault(description, target.Selector)

		if err != nil {
			return nil, err
		}
		steps = append(steps, *restartStep)
	}

	return steps, nil
}

func areExprSelectorsMatchingIdIn(expressionSelectors []ChaosExpressionSelector) error {
	for _, selector := range expressionSelectors {
		if selector.Key != "kurtosistech.com/id" {
			return stacktrace.NewError("i/o latency faults can only be target using pod id: %s", selector.Key)
		}
		if selector.Operator != "In" {
			return stacktrace.NewError("i/o latency faults can only be target using the 'In' operator: %s", selector.Operator)
		}
	}
	return nil
}

func composeIOLatencySteps(targetsSelected []*ChaosTargetSelector, delay *time.Duration, percent int, duration *time.Duration) ([]types.PlanStep, error) {
	var steps []types.PlanStep

	for _, target := range targetsSelected {
		description := fmt.Sprintf("Inject i/o latency on target %s", target.Description)
		err := areExprSelectorsMatchingIdIn(target.Selector)
		if err != nil {
			return nil, err
		}

		// for i/o faults, we need to create a plan step for each individual pod because the fault spec has to say the data path.
		for _, selector := range target.Selector {
			ioLatencySteps, err := buildIOLatencyFault(description, selector, delay, percent, duration)
			if err != nil {
				return nil, err
			}
			steps = append(steps, ioLatencySteps...)
		}
	}

	return steps, nil

}

func composeNetworkLatencySteps(targetsSelected []*ChaosTargetSelector, delay, jitter, duration *time.Duration, correlation int) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	for _, target := range targetsSelected {
		description := fmt.Sprintf("Inject network latency on target %s", target.Description)

		skewStep, err := buildNetworkLatencyFault(description, target.Selector, delay, jitter, duration, correlation)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *skewStep)
	}

	return steps, nil
}

func composePacketDropSteps(targetsSelected []*ChaosTargetSelector, percent int, direction string, duration *time.Duration) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	for _, target := range targetsSelected {
		description := fmt.Sprintf("Inject network latency on target %s", target.Description)

		skewStep, err := buildPacketDropFault(description, target.Selector, percent, direction, duration)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *skewStep)
	}

	return steps, nil
}
