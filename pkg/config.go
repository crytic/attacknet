package pkg

import (
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v2"
	"os"
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

func LoadConfig(path string) (*ConfigParsed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read file config file on path %s", path)
	}
	cfg := Config{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not unmarshal the main config file")
	}

	packageConfig, err := os.ReadFile(cfg.HarnessConfig.NetworkConfigPath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not read the NetworkConfigPath file at %s", cfg.HarnessConfig.NetworkConfigPath)
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
