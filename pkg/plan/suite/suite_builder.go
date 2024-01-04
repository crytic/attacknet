package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"gopkg.in/yaml.v3"
)

func WritePlab(networkConfigPath string, networkConfig []byte) ([]byte, error) {
	nodes, err := network.ParseNetworkConfig(networkConfig)
	if err != nil {
		return nil, err
	}
	skew := "-5m"
	duration := "10m"

	test, err := buildNodeClockSkewTest("clock skew", nodes, skew, duration)
	if err != nil {
		return nil, err
	}
	var a []types.SuiteTest
	tests := append(a, *test)
	_ = tests
	c := types.Config{
		AttacknetConfig: types.AttacknetConfig{
			GrafanaPodName:             "grafana",
			GrafanaPodPort:             "3000",
			WaitBeforeInjectionSeconds: 60,
			ReuseDevnetBetweenRuns:     true,
			ExistingDevnetNamespace:    "kt-ethereum",
			AllowPostFaultInspection:   false,
		},
		HarnessConfig: types.HarnessConfig{
			NetworkPackage:    "github.com/kurtosis-tech/ethereum-package",
			NetworkConfigPath: networkConfigPath,
			NetworkType:       "ethereum",
		},
		TestConfig: types.SuiteTestConfigs{Tests: tests},
	}

	if err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}
