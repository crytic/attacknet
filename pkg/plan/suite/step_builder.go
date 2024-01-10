package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type clientType string

const (
	Execution clientType = "execution"
	Consensus clientType = "consensus"
	Validator clientType = "validator"
)

func convertToNodeIdTag(node *network.Node, client clientType) string {
	switch client {
	case Execution:
		return fmt.Sprintf("el-%d-%s-%s", node.Index, node.Execution.Type, node.Consensus.Type)
	case Consensus:
		return fmt.Sprintf("cl-%d-%s-%s", node.Index, node.Consensus.Type, node.Execution.Type)
	case Validator:
		return fmt.Sprintf("cl-%d-%s-%s-validator", node.Index, node.Consensus.Type, node.Execution.Type)
	default:
		log.Errorf("Unrecognized node type %s", client)
		return ""
	}
}

func composeWaitForFaultCompletionStep() *types.PlanStep {
	return &types.PlanStep{StepType: types.WaitForFaultCompletion, StepDescription: "wait for faults to terminate"}
}

func composeNodeClockSkewPlanSteps(nodesSelected []*ChaosTargetSelector, skew, duration string) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	for _, target := range nodesSelected {
		description := fmt.Sprintf("Inject clock skew on target %s", target.Description)

		skewStep, err := buildClockSkewFault(description, skew, duration, target.Selector)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *skewStep)
	}

	return steps, nil
}

func composeNodeRestartSteps(nodesSelected []*ChaosTargetSelector) ([]types.PlanStep, error) {
	var steps []types.PlanStep

	for _, target := range nodesSelected {
		description := fmt.Sprintf("Restart target %s", target.Description)
		restartStep, err := buildPodRestartFault(description, target.Selector)

		if err != nil {
			return nil, err
		}
		steps = append(steps, *restartStep)
	}

	return steps, nil
}
