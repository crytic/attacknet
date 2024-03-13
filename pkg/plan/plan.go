package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/plan/suite"
	types "attacknet/cmd/pkg/types"
	"gopkg.in/yaml.v3"
)

func BuildPlan(planName string, config *PlannerConfig) error {

	netRefPath, netConfigPath, suiteConfigPath, err := preparePaths(planName)
	if err != nil {
		return err
	}

	nodes, err := network.ComposeNetworkTopology(
		config.Topology,
		config.FaultConfig.TargetClient,
		config.ExecutionClients,
		config.ConsensusClients,
	)
	if err != nil {
		return err
	}

	isExecTarget := config.IsTargetExecutionClient()
	// exclude the bootnode from test targeting
	potentialNodesUnderTest := nodes[1:]
	tests, err := suite.ComposeTestSuite(config.FaultConfig, isExecTarget, potentialNodesUnderTest)
	if err != nil {
		return err
	}

	var attacknetConfig types.AttacknetConfig
	if config.KubernetesNamespace == "" {
		attacknetConfig = types.AttacknetConfig{
			GrafanaPodName:             "grafana",
			GrafanaPodPort:             "3000",
			WaitBeforeInjectionSeconds: uint32(config.FaultConfig.WaitBeforeFirstTest.Seconds()),
			ReuseDevnetBetweenRuns:     true,
			AllowPostFaultInspection:   false,
		}
	} else {
		attacknetConfig = types.AttacknetConfig{
			GrafanaPodName:             "grafana",
			GrafanaPodPort:             "3000",
			WaitBeforeInjectionSeconds: uint32(config.FaultConfig.WaitBeforeFirstTest.Seconds()),
			ReuseDevnetBetweenRuns:     true,
			ExistingDevnetNamespace:    config.KubernetesNamespace,
			AllowPostFaultInspection:   false,
		}
	}

	c := types.Config{
		AttacknetConfig: attacknetConfig,
		HarnessConfig: types.HarnessConfig{
			NetworkPackage:    config.KurtosisPackage,
			NetworkConfigPath: netRefPath,
			NetworkType:       "ethereum",
		},
		TestConfig: types.SuiteTestConfigs{Tests: tests},
	}

	suiteConfig, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	networkConfig, err := SerializeNetworkTopology(nodes, &config.GenesisParams)
	if err != nil {
		return err
	}

	return writePlans(netConfigPath, suiteConfigPath, networkConfig, suiteConfig)
}
