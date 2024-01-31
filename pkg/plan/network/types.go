package network

import "fmt"

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
	NumValKeysPerNode                  int     `yaml:"num_validator_keys_per_node"`
}

type Topology struct {
	BootnodeEL                string  `yaml:"bootnode_el"`
	BootnodeCl                string  `yaml:"bootnode_cl"`
	TargetsAsPercentOfNetwork float32 `yaml:"targets_as_percent_of_network"`
	TargetNodeMultiplier      uint    `yaml:"target_node_multiplier"`
}

type ClientVersion struct {
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	HasSidecar bool   `yaml:"has_sidecar,omitempty"`
}

type ExecutionClient struct {
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
	Execution      *ExecutionClient
	Consensus      *ConsensusClient
	ConsensusVotes int
}

func (n *Node) ToString() string {
	return fmt.Sprintf("#%d %s/%s", n.Index, n.Execution.Type, n.Consensus.Type)
}
