package network

type ExecClient struct {
	Type           string
	Image          string
	ExtraLabels    map[string]string
	CpuRequired    int
	MemoryRequired int
}

type ConsensusClient struct {
	Type                  string
	Image                 string
	HasValidatorSidecar   bool
	ValidatorImage        string
	ExtraLabels           map[string]string
	CpuRequired           int
	MemoryRequired        int
	SidecarCpuRequired    int
	SidecarMemoryRequired int
}

type Node struct {
	Index          int
	Execution      *ExecClient
	Consensus      *ConsensusClient
	ConsensusVotes int
}

// todo: how much of these should we move to the config module?
type EthPackageNetworkParams struct {
	CapellaForkEpoch  int `yaml:"capella_fork_epoch,omitempty"`
	NumValKeysPerNode int `yaml:"num_validator_keys_per_node"`
}

type EthNetConfig struct {
	Participants        []*Participant           `yaml:"participants"`
	NetParams           *EthPackageNetworkParams `yaml:"network_params"`
	AdditionalServices  []string                 `yaml:"additional_services"`
	ParallelKeystoreGen bool                     `yaml:"parallel_keystore_generation"`
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
