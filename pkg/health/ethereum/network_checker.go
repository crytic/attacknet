package ethereum

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	log "github.com/sirupsen/logrus"
	"time"
)
import "attacknet/cmd/pkg/health/types"

type EthNetworkChecker struct {
	kubeClient           *kubernetes.KubeClient
	podsUnderTest        []*chaos_mesh.PodUnderTest
	podsUnderTestLookup  map[string]*chaos_mesh.PodUnderTest
	healthCheckStartTime time.Time
}

func CreateEthNetworkChecker(kubeClient *kubernetes.KubeClient, podsUnderTest []*chaos_mesh.PodUnderTest) *EthNetworkChecker {
	// convert podsUnderTest to a lookup
	podsUnderTestMap := make(map[string]*chaos_mesh.PodUnderTest)

	for _, pod := range podsUnderTest {
		podsUnderTestMap[pod.Name] = pod
	}

	return &EthNetworkChecker{
		podsUnderTest:        podsUnderTest,
		podsUnderTestLookup:  podsUnderTestMap,
		kubeClient:           kubeClient,
		healthCheckStartTime: time.Now(),
	}
}

func (e *EthNetworkChecker) RunAllChecks(ctx context.Context, prevHealthCheckResult *types.HealthCheckResult) (*types.HealthCheckResult, error) {
	execRpcClients, err := e.dialToExecutionClients(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug("Ready to query for health checks")
	latestResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "latest", 5)
	if err != nil {
		return nil, err
	}
	latestArtifact := e.convertResultToArtifact(prevHealthCheckResult.LatestElBlockResult, latestResult)

	finalResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "finalized", 3)
	if err != nil {
		return nil, err
	}
	finalArtifact := e.convertResultToArtifact(prevHealthCheckResult.FinalizedElBlockResult, finalResult)

	log.Debugf("Finalization -> latest lag: %d", latestResult.ConsensusBlock-finalResult.ConsensusBlock)

	results := &types.HealthCheckResult{
		LatestElBlockResult:    latestArtifact,
		FinalizedElBlockResult: finalArtifact,
	}

	return results, nil
}

func (e *EthNetworkChecker) convertResultToArtifact(
	prevArtifact *types.BlockConsensusArtifact,
	result *types.BlockConsensusTestResult) *types.BlockConsensusArtifact {

	timeSinceChecksStarted := time.Since(e.healthCheckStartTime)
	recoveredClients := make(map[string]int)

	if prevArtifact != nil {
		// we only mark clients as recovered if at some point they were failing health checks.
		for client := range prevArtifact.FailingClientsReportedHash {
			if _, stillFailing := result.FailingClientsReportedHash[client]; !stillFailing {
				recoveredClients[client] = int(timeSinceChecksStarted.Seconds())
			}
		}

		for client := range prevArtifact.FailingClientsReportedBlock {
			if _, stillFailing := result.FailingClientsReportedBlock[client]; !stillFailing {
				recoveredClients[client] = int(timeSinceChecksStarted.Seconds())
			}
		}

		// merge previously recovered clients with the new
		for k, v := range prevArtifact.NodeRecoveryTimeSeconds {
			recoveredClients[k] = v
		}
	}

	didUnfaultedNodesNeedToRecover := false
	for client := range recoveredClients {
		if _, wasUnderTest := e.podsUnderTestLookup[client]; !wasUnderTest {
			didUnfaultedNodesNeedToRecover = true
		}
	}

	didUnfaultedNodesFail := false
	for client := range result.FailingClientsReportedBlock {
		if _, wasUnderTest := e.podsUnderTestLookup[client]; !wasUnderTest {
			didUnfaultedNodesFail = true
		}
	}
	for client := range result.FailingClientsReportedHash {
		if _, wasUnderTest := e.podsUnderTestLookup[client]; !wasUnderTest {
			didUnfaultedNodesFail = true
		}
	}

	return &types.BlockConsensusArtifact{
		BlockConsensusTestResult:       result,
		DidUnfaultedNodesFail:          didUnfaultedNodesFail,
		DidUnfaultedNodesNeedToRecover: didUnfaultedNodesNeedToRecover,
		NodeRecoveryTimeSeconds:        recoveredClients,
	}
}
