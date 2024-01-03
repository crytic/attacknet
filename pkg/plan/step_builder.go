package plan

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

func convertToNodeIdTag(validator *network.Validator, client clientType) string {
	switch client {
	case Execution:
		return fmt.Sprintf("el-%d-%s-%s", validator.Index, validator.Execution.Type, validator.Consensus.Type)
	case Consensus:
		return fmt.Sprintf("cl-%d-%s-%s", validator.Index, validator.Consensus.Type, validator.Execution.Type)
	case Validator:
		return fmt.Sprintf("cl-%d-%s-%s-validator", validator.Index, validator.Consensus.Type, validator.Execution.Type)
	default:
		log.Errorf("Unrecognized node type %s", client)
		return ""
	}
}

func buildWaitForFaultCompletionStep() *types.PlanStep {
	return &types.PlanStep{StepType: types.WaitForFaultCompletion, StepDescription: "wait for faults to terminate"}
}

func buildParamsForNodeFault(validator *network.Validator, description string) (elSelector, clSelector, valSelector map[string]string, elDescr, clDescr, valDescr string) {
	elSelector = make(map[string]string)
	elId := convertToNodeIdTag(validator, Execution)
	elSelector["kurtosistech.com/id"] = elId
	elDescr = fmt.Sprintf(description, elId)

	clSelector = make(map[string]string)
	clId := convertToNodeIdTag(validator, Consensus)
	clSelector["kurtosistech.com/id"] = clId
	clDescr = fmt.Sprintf(description, clId)

	if validator.Consensus.HasValidatorSidecar {
		valSelector = make(map[string]string)
		valId := convertToNodeIdTag(validator, Validator)
		valSelector["kurtosistech.com/id"] = valId
		valDescr = fmt.Sprintf(description, valId)
	} else {
		valSelector = make(map[string]string)
		valDescr = "n/a"
	}
	return
}

func buildNodeClockSkewPlanSteps(validator *network.Validator, skew, duration string) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	descriptionGeneric := "Inject clock skew on %s"
	elSelector, clSelector, valSelector, elDescr, clDescr, valDescr := buildParamsForNodeFault(validator, descriptionGeneric)
	isolateELStep, err := buildClockSkewFault(elDescr, skew, duration, elSelector)
	if err != nil {
		return nil, err
	}
	steps = append(steps, *isolateELStep)

	isolateClStep, err := buildClockSkewFault(clDescr, skew, duration, clSelector)
	if err != nil {
		return nil, err
	}
	steps = append(steps, *isolateClStep)

	if validator.Consensus.HasValidatorSidecar {
		isolateValStep, err := buildClockSkewFault(valDescr, skew, duration, valSelector)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *isolateValStep)
	}

	return steps, nil
}

func buildNodeRestartSteps(validator *network.Validator) ([]types.PlanStep, error) {
	var steps []types.PlanStep
	descriptionGeneric := "Restart pod %s"
	elSelector, clSelector, valSelector, elDescr, clDescr, valDescr := buildParamsForNodeFault(validator, descriptionGeneric)
	isolateELStep, err := buildPodRestartFault(elDescr, elSelector)
	if err != nil {
		return nil, err
	}
	steps = append(steps, *isolateELStep)

	isolateClStep, err := buildPodRestartFault(clDescr, clSelector)
	if err != nil {
		return nil, err
	}
	steps = append(steps, *isolateClStep)

	if validator.Consensus.HasValidatorSidecar {
		isolateValStep, err := buildPodRestartFault(valDescr, valSelector)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *isolateValStep)
	}

	return steps, nil
}
