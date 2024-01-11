package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/plan/suite"
)

type PlannerConfig struct {
	ExecutionClients    []network.ClientVersion         `yaml:"execution"`
	ConsensusClients    []network.ClientVersion         `yaml:"consensus"`
	GenesisParams       network.GenesisConfig           `yaml:"network_params"`
	KurtosisPackage     string                          `yaml:"kurtosis_package"`
	KubernetesNamespace string                          `yaml:"kubernetes_namespace"`
	FaultConfig         suite.PlannerFaultConfiguration `yaml:"fault_config"`
}

func (c *PlannerConfig) IsTargetExecutionClient() bool {
	for _, execClient := range c.ExecutionClients {
		if execClient.Name == c.FaultConfig.TargetClient {
			return true
		}
	}
	return false
}

func (c *PlannerConfig) IsTargetConsensusClient() bool {
	for _, consClient := range c.ConsensusClients {
		if consClient.Name == c.FaultConfig.TargetClient {
			return true
		}
	}
	return false
}

type EthKurtosisConfig struct {
	Participants        []*Participant        `yaml:"participants"`
	NetParams           network.GenesisConfig `yaml:"network_params"`
	AdditionalServices  []string              `yaml:"additional_services"`
	ParallelKeystoreGen bool                  `yaml:"parallel_keystore_generation"`
	Persistent          bool                  `yaml:"persistent"`
}

type Participant struct {
	ElClientType  string `yaml:"el_client_type"`
	ElClientImage string `yaml:"el_client_image"`

	ClClientType  string `yaml:"cl_client_type"`
	ClClientImage string `yaml:"cl_client_image"`

	ElMinCpu    int `yaml:"el_min_cpu"`
	ElMaxCpu    int `yaml:"el_max_cpu"`
	ElMinMemory int `yaml:"el_min_mem"`
	ElMaxMemory int `yaml:"el_max_mem"`

	ClMinCpu    int `yaml:"bn_min_cpu"`
	ClMaxCpu    int `yaml:"bn_max_cpu"`
	ClMinMemory int `yaml:"bn_min_mem"`
	ClMaxMemory int `yaml:"bn_max_mem"`

	ValMinCpu    int `yaml:"v_min_cpu,omitempty"`
	ValMaxCpu    int `yaml:"v_max_cpu,omitempty"`
	ValMinMemory int `yaml:"v_min_mem,omitempty"`
	ValMaxMemory int `yaml:"v_max_mem,omitempty"`

	Count int `yaml:"count"`
}
