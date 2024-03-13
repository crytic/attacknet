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
	beaconRpcClients, err := e.dialToBeaconClients(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug("Ready to query for health checks")
	latestElResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "latest", 15)
	if err != nil {
		return nil, err
	}
	latestElArtifact := e.convertResultToArtifact(prevHealthCheckResult.LatestElBlockResult, latestElResult)

	finalElResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "finalized", 3)
	if err != nil {
		return nil, err
	}
	finalElArtifact := e.convertResultToArtifact(prevHealthCheckResult.FinalizedElBlockResult, finalElResult)

	log.Debugf("Finalization -> latest lag: %d", latestElResult.ConsensusBlock-finalElResult.ConsensusBlock)

	latestClResult, err := e.getBeaconClientConsensus(ctx, beaconRpcClients, "head", 15)
	if err != nil {
		return nil, err
	}
	latestClArtifact := e.convertResultToArtifact(prevHealthCheckResult.LatestClBlockResult, latestClResult)

	finalClResult, err := e.getBeaconClientConsensus(ctx, beaconRpcClients, "finalized", 3)
	if err != nil {
		return nil, err
	}
	finalClArtifact := e.convertResultToArtifact(prevHealthCheckResult.FinalizedClBlockResult, finalClResult)

	results := &types.HealthCheckResult{
		LatestElBlockResult:    latestElArtifact,
		FinalizedElBlockResult: finalElArtifact,
		LatestClBlockResult:    latestClArtifact,
		FinalizedClBlockResult: finalClArtifact,
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
