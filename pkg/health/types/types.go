package types

import (
	chaosMesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/types"
	"context"
)

type HealthChecker interface {
	RunChecks(ctx context.Context) (bool, error)
	PopFinalResult() interface{}
}

type BlockConsensusTestResult struct {
	ConsensusBlock              uint64            `yaml:"consensus_block"`
	ConsensusHash               string            `yaml:"consensus_hash"`
	FailingClientsReportedBlock map[string]uint64 `yaml:"failing_clients_reported_block"`
	FailingClientsReportedHash  map[string]string `yaml:"failing_clients_reported_hash"`
}

type BlockConsensusArtifact struct {
	*BlockConsensusTestResult      `yaml:",inline"`
	DidUnfaultedNodesFail          bool           `yaml:"did_unfaulted_nodes_fail"`
	DidUnfaultedNodesNeedToRecover bool           `yaml:"did_unfaulted_nodes_need_to_recover"`
	NodeRecoveryTimeSeconds        map[string]int `yaml:"node_recovery_time_seconds"`
}

type ArtifactSerializer interface {
	AddHealthCheckResult(interface{}, []*chaosMesh.PodUnderTest, types.SuiteTest) error
	SerializeArtifacts() ([]byte, error)
}
