package suite

import (
	planTypes "attacknet/cmd/pkg/plan/types"
	"attacknet/cmd/pkg/types"
	"gopkg.in/yaml.v3"
)

func ComposeAndSerializeTestSuite(
	faultConfig planTypes.PlannerFaultConfiguration,
	networkConfigPath string,
	nodes []*planTypes.Node) ([]byte, error) {

	var tests []types.SuiteTest

	//slice by targeting (client, node)
	// -> creates targeting lambdas
	//slice by attack size
	// -> takes nodes[], returns target selectors. count reduced as aliasing overlaps will cause dupes

	// slice by intensity
	// -> creates []intensities

	_ = tests
	return nil, nil
}

func WritePlab(networkConfigPath string, nodes []*planTypes.Node) ([]byte, error) {
	skew := "-5m"
	duration := "1m"
	criteriaLambda := createDualClientTargetCriteria("reth", "teku")
	targetSelectors, err := BuildTargetSelectors(nodes, planTypes.AttackAll, criteriaLambda, impactNode)
	if err != nil {
		return nil, err
	}

	test, err := buildNodeClockSkewTest("clock skew", targetSelectors, skew, duration)
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
			WaitBeforeInjectionSeconds: 0,
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
