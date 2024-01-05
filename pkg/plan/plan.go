package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/plan/suite"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"path/filepath"
)

func preparePaths(testName string) (netRefPath, netConfigPath, planConfigPath string, err error) {
	dir, err := os.Getwd()
	// initialize to empty string for error cases
	netConfigPath = ""
	planConfigPath = ""
	if err != nil {
		return
	}

	netRefPath = fmt.Sprintf("plan/%s.yaml", testName)
	networkConfigName := fmt.Sprintf("network-configs/%s", netRefPath)
	netConfigPath = filepath.Join(dir, networkConfigName)
	if _, err = os.Stat(netConfigPath); err == nil {
		// delete file
		err = os.Remove(netConfigPath)
		if err != nil {
			err = stacktrace.Propagate(err, "unable to remove file")
			return
		}
	}

	suiteName := fmt.Sprintf("test-suites/plan/%s.yaml", testName)
	planConfigPath = filepath.Join(dir, suiteName)
	if _, err = os.Stat(planConfigPath); err == nil {
		// delete file
		err = os.Remove(planConfigPath)
		if err != nil {
			err = stacktrace.Propagate(err, "unable to remove file")
			return
		}
	}
	err = nil
	return
}

func writePlans(netConfigPath, suiteConfigPath string, netConfig, suiteConfig []byte) error {
	f, err := os.Create(netConfigPath)
	if err != nil {
		return stacktrace.Propagate(err, "cannot open network config path %s", netConfigPath)
	}
	_, err = f.Write(netConfig)
	if err != nil {
		return stacktrace.Propagate(err, "could not write network config to file")
	}

	err = f.Close()
	if err != nil {
		return stacktrace.Propagate(err, "could not close network config file")
	}

	f, err = os.Create(suiteConfigPath)
	if err != nil {
		return stacktrace.Propagate(err, "cannot open suite config path %s", suiteConfigPath)
	}
	_, err = f.Write(suiteConfig)
	if err != nil {
		return stacktrace.Propagate(err, "could not write suite config to file")
	}

	err = f.Close()
	if err != nil {
		return stacktrace.Propagate(err, "could not close suite config file")
	}

	return nil
}

func BuildPlan() error {
	testName := "test"

	netRefPath, netConfigPath, suiteConfigPath, err := preparePaths(testName)
	if err != nil {
		return err
	}

	netConfig, err := network.BuildExecTesterNetwork("reth")
	if err != nil {
		return err
	}

	suiteConfig, err := suite.WritePlab(netRefPath, netConfig)
	if err != nil {
		return err
	}

	return writePlans(netConfigPath, suiteConfigPath, netConfig, suiteConfig)
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

			syncing faults
			-> restart node, force to sync. inject fault while syncing. this impacts checkpoint sync probably too.

			packet corruption


		each test builder needs a way to reject input corpus
		eventually we'll want a way to block known bad inputs (ie: lodestar doesnt seem to re-establish peers correctly)
		anotehr example:

		actual tasks:
		- implement plan builder for each concept

			selector := buildParamsForNodeFault(node)
	*/
	//return nil
}
