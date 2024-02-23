package suite

import "time"

type TargetingSpec string

const (
	TargetMatchingNode   TargetingSpec = "MatchingNode"
	TargetMatchingClient TargetingSpec = "MatchingClient"
)

var TargetingSpecs = map[TargetingSpec]bool{
	TargetMatchingNode:   true,
	TargetMatchingClient: true,
}

var TargetingSpecList = []TargetingSpec{
	TargetMatchingNode,
	TargetMatchingClient,
}

type AttackSize string

const (
	AttackOne           AttackSize = "AttackOneMatching"
	AttackAll           AttackSize = "AttackAllMatching"
	AttackMinority      AttackSize = "AttackMinorityMatching"      // scope will be 0<x<33.333
	AttackSuperminority AttackSize = "AttackSuperminorityMatching" // scope will be 33.333 < x < 50
	AttackMajority      AttackSize = "AttackMajorityMatching"      // scope will be 50 < x < 66.6
	AttackSupermajority AttackSize = "AttackSupermajorityMatching" // scope will be 66.66 < x < 100
	// if we want to add 33.33%, 66.66%, and 50%, there needs to be logic to verify the network can be split into such
	// fractions, otherwise using the number in target selection may be misleading.
)

var AttackSizes = map[AttackSize]bool{
	AttackOne:           true,
	AttackAll:           true,
	AttackMinority:      true,
	AttackSuperminority: true,
	AttackMajority:      true,
	AttackSupermajority: true,
}

var AttackSizesList = []AttackSize{
	AttackOne,
	AttackAll,
	AttackMinority,
	AttackSuperminority,
	AttackMajority,
	AttackSupermajority,
}

type FaultTypeEnum string

const (
	FaultClockSkew        FaultTypeEnum = "ClockSkew"
	FaultContainerRestart FaultTypeEnum = "RestartContainers"
	FaultIOLatency        FaultTypeEnum = "IOLatency"
	FaultNetworkLatency   FaultTypeEnum = "NetworkLatency"
	FaultPacketLoss       FaultTypeEnum = "PacketLoss"
)

var FaultTypes = map[FaultTypeEnum]bool{
	FaultClockSkew:        true,
	FaultContainerRestart: true,
	FaultIOLatency:        true,
	FaultNetworkLatency:   true,
}

var FaultTypesList = []FaultTypeEnum{
	FaultClockSkew,
	FaultContainerRestart,
	FaultIOLatency,
	FaultNetworkLatency,
	FaultPacketLoss,
}

type PlannerFaultConfiguration struct {
	FaultType             FaultTypeEnum       `yaml:"fault_type"`
	TargetClient          string              `yaml:"target_client"`
	WaitBeforeFirstTest   time.Duration       `yaml:"wait_before_first_test"`
	FaultConfigDimensions []map[string]string `yaml:"fault_config_dimensions"`
	TargetingDimensions   []TargetingSpec     `yaml:"fault_targeting_dimensions"`
	AttackSizeDimensions  []AttackSize        `yaml:"fault_attack_size_dimensions"`
}
