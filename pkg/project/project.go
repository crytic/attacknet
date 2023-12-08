package project

import (
	"fmt"
	"os"
	"path/filepath"
)
import "errors"

var suiteDirectory = "test-suites"
var networkConfigDirectory = "network-configs"

func projectAlreadyExists(suitePath string, configPath string) bool {
	if _, err := os.Stat(suitePath); !os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		return true
	}

	return false
}

func initializeDirectoriesAndFiles(suitePath string, configPath string) error {
	err := os.Mkdir(suitePath, 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir(configPath, 0755)
	if err != nil {
		return err
	}

	fmt.Print("Todo: we need to create sample files for the suite/config dirs. They will be empty until then.")
	return nil
}

func InitializeProject(path string, force bool) error {
	suitePath := filepath.Join(path, suiteDirectory)
	configPath := filepath.Join(path, networkConfigDirectory)

	if !force {
		exists := projectAlreadyExists(suitePath, configPath)
		if exists {
			errStr := fmt.Sprintf("A project already exsts on %s. Re-run with --force to overwrite.", path)
			fmt.Print(errStr)
			return errors.New(errStr)
		}
	}

	return initializeDirectoriesAndFiles(suitePath, configPath)
}
