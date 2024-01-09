package plan

import (
	planTypes "attacknet/cmd/pkg/plan/types"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"os"
)

func validatePlannerFaultConfiguration(c planTypes.PlannerConfig) error {
	// fault type
	_, ok := planTypes.FaultTypes[c.FaultConfig.FaultType]
	if !ok {
		faults := make([]planTypes.FaultTypeEnum, 0, len(planTypes.FaultTypes))
		for k := range planTypes.FaultTypes {
			faults = append(faults, k)
		}
		return stacktrace.NewError("the fault type '%s' is not supported. Supported faults: %v", c.FaultConfig.FaultType, faults)
	}

	// target client
	// todo

	// intensity domains
	// todo

	// targeting dimensions
	for _, spec := range c.FaultConfig.TargetingDimensions {
		_, ok := planTypes.TargetingSpecs[spec]
		if !ok {
			specs := make([]planTypes.TargetingSpec, 0, len(planTypes.TargetingSpecs))
			for k := range planTypes.TargetingSpecs {
				specs = append(specs, k)
			}
			return stacktrace.NewError("the fault targeting dimension %s is not supported. Supported dimensions: %v", spec, specs)
		}
	}

	// attack size dimensions
	for _, attackSize := range c.FaultConfig.AttackSizeDimensions {
		_, ok := planTypes.AttackSizes[attackSize]
		if !ok {
			sizes := make([]planTypes.AttackSize, 0, len(planTypes.AttackSizes))
			for k := range planTypes.AttackSizes {
				sizes = append(sizes, k)
			}
			return stacktrace.NewError("the attack size dimension %s is not supported. Supported dimensions: %v", attackSize, sizes)
		}
	}

	// target client
	if c.FaultConfig.TargetClient != "all" {
		if !c.IsTargetExecutionClient() && !c.IsTargetConsensusClient() {
			return stacktrace.NewError("target_client %s is not defined in the execution/consensus client configuration", c.FaultConfig.TargetClient)
		}
	}

	return nil
}

func LoadPlannerConfigFromPath(path string) (*planTypes.PlannerConfig, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, stacktrace.Propagate(err, "could not planner config on path %s", path)
	}

	var config planTypes.PlannerConfig
	err = yaml.Unmarshal(bs, &config)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to unmarshal planner config from %s", path)
	}

	err = validatePlannerFaultConfiguration(config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
