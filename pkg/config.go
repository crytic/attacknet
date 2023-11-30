package pkg

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"

	"github.com/kurtosis-tech/stacktrace"
	yaml "gopkg.in/yaml.v3"
)

type AttacknetConfig struct {
	GrafanaPodName             string `yaml:"grafanaPodName"`
	GrafanaPodPort             string `yaml:"grafanaPodPort"`
	WaitBeforeInjectionSeconds uint32 `yaml:"waitBeforeInjectionSeconds"`
	ReuseDevnetBetweenRuns     bool   `yaml:"reuseDevnetBetweenRuns"`
	ExistingDevnetNamespace    string `yaml:"existingDevnetNamespace"`
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
	Name      string                 `yaml:"testName"`
	FaultSpec map[string]interface{} `yaml:"chaosFaultSpec"`
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

func defaultConfig() *Config {
	cfg := Config{
		AttacknetConfig: AttacknetConfig{
			WaitBeforeInjectionSeconds: 0,
			ExistingDevnetNamespace:    "",
			ReuseDevnetBetweenRuns:     false,
		},
	}
	return &cfg
}

func LoadSuiteConfigFromName(suiteName string) (*ConfigParsed, error) {
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

func loadSuiteFromPath(path string) (*ConfigParsed, error) {
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
