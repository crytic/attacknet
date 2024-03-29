package ethereum

import (
	chaosmesh "attacknet/cmd/pkg/chaos-mesh"
	healthTypes "attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	"time"
)

var (
	_ healthTypes.ArtifactSerializer = (*artifactSerializer)(nil)
	_ healthTypes.HealthChecker      = (*healthChecker)(nil)
)

type healthChecker struct {
	kubeClient            *kubernetes.KubeClient
	podsUnderTest         []*chaosmesh.PodUnderTest
	podsUnderTestLookup   map[string]*chaosmesh.PodUnderTest
	healthCheckStartTime  time.Time
	prevHealthCheckResult *healthCheckResult
}

type artifactSerializer struct {
	artifacts []*testArtifact
}

type testArtifact struct {
	TestDescription    string             `yaml:"test_description"`
	ContainersTargeted []string           `yaml:"fault_injection_targets"`
	TestPassed         bool               `yaml:"test_passed"`
	HealthResult       *healthCheckResult `yaml:"health_check_results"`
}

type healthCheckResult struct {
	LatestElBlockResult    *healthTypes.BlockConsensusArtifact `yaml:"latest_el_block_health_result"`
	FinalizedElBlockResult *healthTypes.BlockConsensusArtifact `yaml:"finalized_el_block_health_result"`
	LatestClBlockResult    *healthTypes.BlockConsensusArtifact `yaml:"latest_cl_block_health_result"`
	FinalizedClBlockResult *healthTypes.BlockConsensusArtifact `yaml:"finalized_cl_block_health_result"`
}
