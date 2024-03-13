package types

import "context"

type GenericNetworkChecker interface {
	RunAllChecks(context.Context, *HealthCheckResult) (*HealthCheckResult, error)
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

type HealthCheckResult struct {
	LatestElBlockResult    *BlockConsensusArtifact `yaml:"latest_el_block_health_result"`
	FinalizedElBlockResult *BlockConsensusArtifact `yaml:"finalized_el_block_health_result"`
	LatestClBlockResult    *BlockConsensusArtifact `yaml:"latest_cl_block_health_result"`
	FinalizedClBlockResult *BlockConsensusArtifact `yaml:"finalized_cl_block_health_result"`
}
