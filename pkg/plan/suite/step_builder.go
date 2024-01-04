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

func buildWaitForFaultCompletionStep() *types.PlanStep {
	return &types.PlanStep{StepType: types.WaitForFaultCompletion, StepDescription: "wait for faults to terminate"}
}

func buildParamsForNodeFault(node *network.Node) (selector ExpressionSelector) {
	targets := []string{}

	elId := convertToNodeIdTag(node, Execution)
	targets = append(targets, elId)

	clId := convertToNodeIdTag(node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := convertToNodeIdTag(node, Validator)
		targets = append(targets, valId)
	}

	selector = ExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   targets,
	}
	return
}

func buildNodeClockSkewPlanSteps(node *network.Node, skew, duration string) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	descriptionGeneric := fmt.Sprintf("Inject clock skew on node #%d", node.Index)
	selector := buildParamsForNodeFault(node)

	skewStep, err := buildClockSkewFault(descriptionGeneric, skew, duration, []ExpressionSelector{selector})
	if err != nil {
		return nil, err
	}
	steps = append(steps, *skewStep)

	return steps, nil
}

func buildNodeRestartSteps(node *network.Node) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	descriptionGeneric := "Restart pod %s"
	selector := buildParamsForNodeFault(node)

	restartStep, err := buildPodRestartFault(descriptionGeneric, []ExpressionSelector{selector})

	if err != nil {
		return nil, err
	}
	steps = append(steps, *restartStep)

	return steps, nil
}
