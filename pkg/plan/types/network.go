package types

// todo: how much of these should we move to the types module?
type GenesisConfig struct {
	PreregisteredValidatorKeysMnemonic *string `yaml:"preregistered_validator_keys_mnemonic,omitempty"`
	PreregisteredValidatorCount        *int    `yaml:"preregistered_validator_count,omitempty"`
	NetworkId                          *int    `yaml:"network_id,omitempty"`
	DepositContractAddress             *string `yaml:"deposit_contract_address,omitempty"`
	SecondsPerSlot                     *int    `yaml:"seconds_per_slot,omitempty"`
	GenesisDelay                       *int    `yaml:"genesis_delay,omitempty"`
	MaxChurn                           *uint64 `yaml:"max_churn,omitempty"`
	EjectionBalance                    *uint64 `yaml:"ejection_balance,omitempty"`
	Eth1FollowDistance                 *int    `yaml:"eth1_follow_distance,omitempty"`
	CapellaForkEpoch                   *int    `yaml:"capella_fork_epoch,omitempty"`
	DenebForkEpoch                     *int    `yaml:"deneb_fork_epoch,omitempty"`
	ElectraForkEpoch                   *int    `yaml:"electra_fork_epoch,omitempty"`

	NumValKeysPerNode int `yaml:"num_validator_keys_per_node"`
}

type EthNetConfig struct {
	Participants        []*Participant `yaml:"participants"`
	NetParams           GenesisConfig  `yaml:"network_params"`
	AdditionalServices  []string       `yaml:"additional_services"`
	ParallelKeystoreGen bool           `yaml:"parallel_keystore_generation"`
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

type ClientVersion struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	HasSidecar bool   `yaml:"has_sidecar,omitempty"`
}
