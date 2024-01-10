package types

import (
	"fmt"
)

type PlannerConfig struct {
	ExecutionClients    []ClientVersion           `yaml:"execution"`
	ConsensusClients    []ClientVersion           `yaml:"consensus"`
	GenesisParams       GenesisConfig             `yaml:"network_params"`
	KurtosisPackage     string                    `yaml:"kurtosis_package"`
	KubernetesNamespace string                    `yaml:"kubernetes_namespace"`
	FaultConfig         PlannerFaultConfiguration `yaml:"fault_config"`
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

type FaultTypeEnum string

const (
	FaultClockSkew        FaultTypeEnum = "ClockSkew"
	FaultContainerRestart FaultTypeEnum = "RestartContainers"
)

var FaultTypes = map[FaultTypeEnum]bool{
	FaultClockSkew:        true,
	FaultContainerRestart: true,
}

var _ = FaultTypes

type PlannerFaultConfiguration struct {
	FaultType             FaultTypeEnum       `yaml:"fault_type"`
	TargetClient          string              `yaml:"target_client"`
	FaultConfigDimensions []map[string]string `yaml:"fault_config_dimensions"`
	TargetingDimensions   []TargetingSpec     `yaml:"fault_targeting_dimensions"`
	AttackSizeDimensions  []AttackSize        `yaml:"fault_attack_size_dimensions"`
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
