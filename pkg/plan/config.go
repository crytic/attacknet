package plan

import (
	"attacknet/cmd/pkg/plan/suite"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"os"
)

func validatePlannerFaultConfiguration(c PlannerConfig) error {
	// fault type
	_, ok := suite.FaultTypes[c.FaultConfig.FaultType]
	if !ok {
		return stacktrace.NewError("the fault type '%s' is not supported. Supported faults: %v", c.FaultConfig.FaultType, suite.FaultTypesList)
	}

	// target client
	// todo

	// intensity domains
	// todo

	// targeting dimensions
	for _, spec := range c.FaultConfig.TargetingDimensions {
		_, ok := suite.TargetingSpecs[spec]
		if !ok {
			return stacktrace.NewError("the fault targeting dimension %s is not supported. Supported dimensions: %v", spec, suite.TargetingSpecList)
		}
	}

	// attack size dimensions
	for _, attackSize := range c.FaultConfig.AttackSizeDimensions {
		_, ok := suite.AttackSizes[attackSize]
		if !ok {
			return stacktrace.NewError("the attack size dimension %s is not supported. Supported dimensions: %v", attackSize, suite.AttackSizesList)
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

func LoadPlannerConfigFromPath(path string) (*PlannerConfig, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, stacktrace.Propagate(err, "could not planner config on path %s", path)
	}

	var config PlannerConfig
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
