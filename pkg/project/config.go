package project

import (
	"attacknet/cmd/pkg/types"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

func defaultConfig() *types.Config {
	cfg := types.Config{
		AttacknetConfig: types.AttacknetConfig{
			WaitBeforeInjectionSeconds: 0,
			ExistingDevnetNamespace:    "",
			ReuseDevnetBetweenRuns:     true,
			AllowPostFaultInspection:   true,
		},
	}
	return &cfg
}

func LoadSuiteConfigFromName(suiteName string) (*types.ConfigParsed, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	suiteName = suiteName + ".yaml"
	suiteFilePath := filepath.Join(dir, suiteDirectory, suiteName)

	log.Infof("Loading test suite from %s", suiteFilePath)
	cfg, err := loadSuiteFromPath(suiteFilePath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadSuiteFromPath(path string) (*types.ConfigParsed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read test suite on %s. is the project initialized?", path)
	}
	cfg := defaultConfig()
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not unmarshal the suite definition file")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not get working directory")
	}
	networkConfigPathFull := filepath.Join(cwd, networkConfigDirectory, cfg.HarnessConfig.NetworkConfigPath)
	log.Infof("Loading kurtosis network configuration from %s", networkConfigPathFull)
	packageConfig, err := os.ReadFile(networkConfigPathFull)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read the NetworkConfigPath file at %s", networkConfigPathFull)
	}

	cfgParsed := &types.ConfigParsed{
		AttacknetConfig: cfg.AttacknetConfig,
		HarnessConfig: types.HarnessConfigParsed{
			NetworkType:    cfg.HarnessConfig.NetworkType,
			NetworkPackage: cfg.HarnessConfig.NetworkPackage,
			NetworkConfig:  packageConfig,
		},
		TestConfig: cfg.TestConfig,
	}

	return cfgParsed, nil
}
