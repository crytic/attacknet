package pkg

import (
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

type AttacknetConfig struct {
	GrafanaPodName string `yaml:"grafanaPodName"`
	GrafanaPodPort string `yaml:"grafanaPodPort"`
	Parallelism    uint32 `yaml:"parallelism"`
}

type HarnessConfig struct {
	NetworkPackage    string `yaml:"networkPackage"`
	NetworkConfigPath string `yaml:"networkConfig"`
}

type HarnessConfigParsed struct {
	NetworkPackage string
	NetworkConfig  []byte
}

type TestConfig struct {
	Name        string      `yaml:"testName"`
	TargetRegex string      `yaml:"targetPodRegex"`
	FaultType   string      `yaml:"faultType"`
	FaultSpec   interface{} `yaml:"faultSpec"`
}

type Config struct {
	AttacknetConfig AttacknetConfig `yaml:"attacknetConfig"`
	HarnessConfig   HarnessConfig   `yaml:"harnessConfig"`
	Tests           []TestConfig    `yaml:"tests"`
}

type ConfigParsed struct {
	AttacknetConfig AttacknetConfig
	HarnessConfig   HarnessConfigParsed
	Tests           []TestConfig
}

func LoadTestSuite(path string) (*ConfigParsed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read test suite on %s. is the project initialized?", path)
	}
	cfg := Config{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not unmarshal the suite definition file")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not get working directory")
	}
	networkConfigPathFull := filepath.Join(cwd, networkConfigDirectory, cfg.HarnessConfig.NetworkConfigPath)

	packageConfig, err := os.ReadFile(networkConfigPathFull)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read the NetworkConfigPath file at %s", networkConfigPathFull)
	}

	cfgParsed := &ConfigParsed{
		cfg.AttacknetConfig,
		HarnessConfigParsed{
			cfg.HarnessConfig.NetworkPackage,
			packageConfig,
		},
		cfg.Tests,
	}

	return cfgParsed, nil
}
