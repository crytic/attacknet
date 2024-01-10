package plan

import (
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
		return stacktrace.Propagate(err, "cannot open network types path %s", netConfigPath)
	}
	_, err = f.Write(netConfig)
	if err != nil {
		return stacktrace.Propagate(err, "could not write network types to file")
	}

	err = f.Close()
	if err != nil {
		return stacktrace.Propagate(err, "could not close network types file")
	}

	f, err = os.Create(suiteConfigPath)
	if err != nil {
		return stacktrace.Propagate(err, "cannot open suite types path %s", suiteConfigPath)
	}
	_, err = f.Write(suiteConfig)
	if err != nil {
		return stacktrace.Propagate(err, "could not write suite types to file")
	}

	err = f.Close()
	if err != nil {
		return stacktrace.Propagate(err, "could not close suite types file")
	}

	return nil
}
