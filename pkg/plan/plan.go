package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

func writePlan(networkConfigPath, suiteConfigPath string) error {
	skew := "-10m"
	duration := "20m"
	node := &network.Validator{
		Index: 4,
		Consensus: &network.ConsensusClientConf{
			Type:                "prysm",
			HasValidatorSidecar: true,
		},
		Execution: &network.ExecClientConf{
			Type: "reth",
		},
	}
	nodes := make([]*network.Validator, 1)
	nodes[0] = node

	test, err := buildNodeClockSkewTest("clock skew", nodes, skew, duration)
	if err != nil {
		return err
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
			ExistingDevnetNamespace:    "kt-ethereum2",
			AllowPostFaultInspection:   false,
		},
		HarnessConfig: types.HarnessConfig{
			NetworkPackage:    "github.com/kurtosis-tech/ethereum-package",
			NetworkConfigPath: "reth.yaml",
			NetworkType:       "ethereum",
		},
		TestConfig: types.SuiteTestConfigs{Tests: tests},
	}

	err = os.Remove("test-suites/plan/reth-reorg.yaml")
	if err != nil {
		return err
	}
	f, err := os.Create("test-suites/plan/reth-reorg.yaml")
	if err != nil {
		return err
	}

	marsh, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = f.Write(marsh)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func BuildPlan() error {

	// clean suite plan dir
	// cleann etwork config dir

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	suiteName := "test-suites/plan/reth-reorg.yaml"
	suiteFilePath := filepath.Join(dir, suiteName)
	if _, err := os.Stat(suiteFilePath); err == nil {
		// delete file
		err = os.Remove(suiteFilePath)
		if err != nil {
			return err
		}
	}

	networkConfigName := "network-configs/plan/reth-reorg.yaml"
	networkFilePath := filepath.Join(dir, networkConfigName)
	if _, err := os.Stat(networkFilePath); err == nil {
		// delete file
		err = os.Remove(networkFilePath)
		if err != nil {
			return err
		}
	}

	return writePlan(networkFilePath, suiteFilePath)
	/*
				run time delay on various el/cl combos
				-> each target exists in the same suite/network

				run time delay on group of el-cl nodes that use the same CL or EL
				-> network minority
				-> 33+ but less than 66%

				re-org on group of el-cl nodes that use the same CL or EL

			there's two steps, identifying targets, and creating the manifest for the target/test config

			targeting criteria types:
			- percentages of the validator set (32, 33, 34, 50, 65)%
			- subcategories: by node vs. by client
			- target by client
				- a specific node containing an instance of the client
				- all nodes containing an instance of the client
				- a specific instance of the client
				- all instances of the client
				- subcategories: target node or target client by criterion


			clock skew
			- extra varies:
				- clock skew nodes by EL
				- clock skew nodes by CL
			- criterion: percentage(client, node), target by client(client, node)


			restarts
			- these restarts require resync
			- criterion: percentages(client, node), target by client(client, node)

			network bandwidth
			- extra varies:
				- the amount of bandwidth
				- whether the constraint is EL<-CL or node <-> network
			- percentages
			- client criterion (although not all client selections will be valid)

			network split
			- percentages
			- client criterion

			packet drop
			- extra varies: loss pct, correlation

			latency
			- extra varies: latency amount, correlation
			- percentages (although includes 100%)
			- clients (both type?)

			packet corruption


		each test builder needs a way to reject input corpus
		eventually we'll want a way to block known bad inputs (ie: lodestar doesnt seem to re-establish peers correctly)

		actual tasks:
		- implement plan builder for each concept

	*/
	//return nil
}
