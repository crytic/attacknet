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
	DisablePeerScoring  bool                  `yaml:"disable_peer_scoring"`
}

type Participant struct {
	ElClientType  string `yaml:"el_type"`
	ElClientImage string `yaml:"el_image"`

	ClClientType  string `yaml:"cl_type"`
	ClClientImage string `yaml:"cl_image"`

	ElMinCpu    int `yaml:"el_min_cpu"`
	ElMaxCpu    int `yaml:"el_max_cpu"`
	ElMinMemory int `yaml:"el_min_mem"`
	ElMaxMemory int `yaml:"el_max_mem"`

	ClMinCpu    int `yaml:"cl_min_cpu"`
	ClMaxCpu    int `yaml:"cl_max_cpu"`
	ClMinMemory int `yaml:"cl_min_mem"`
	ClMaxMemory int `yaml:"cl_max_mem"`

	ValMinCpu    int `yaml:"vc_min_cpu,omitempty"`
	ValMaxCpu    int `yaml:"vc_max_cpu,omitempty"`
	ValMinMemory int `yaml:"vc_min_mem,omitempty"`
	ValMaxMemory int `yaml:"vc_max_mem,omitempty"`

	Count int `yaml:"count"`
}
