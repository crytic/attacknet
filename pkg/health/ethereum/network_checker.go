package ethereum

import (
	chaosmesh "attacknet/cmd/pkg/chaos-mesh"
	healthTypes "attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

func CreateEthNetworkChecker(
	kubeClient *kubernetes.KubeClient,
	podsUnderTest []*chaosmesh.PodUnderTest,
) healthTypes.HealthChecker {
	// convert podsUnderTest to a lookup
	podsUnderTestMap := make(map[string]*chaosmesh.PodUnderTest)

	for _, pod := range podsUnderTest {
		podsUnderTestMap[pod.Name] = pod
	}

	return &healthChecker{
		podsUnderTest:         podsUnderTest,
		podsUnderTestLookup:   podsUnderTestMap,
		kubeClient:            kubeClient,
		healthCheckStartTime:  time.Now(),
		prevHealthCheckResult: &healthCheckResult{},
	}
}

func (e *healthChecker) RunChecks(ctx context.Context) (bool, error) {
	execRpcClients, err := e.dialToExecutionClients(ctx)
	if err != nil {
		return false, err
	}
	beaconRpcClients, err := e.dialToBeaconClients(ctx)
	if err != nil {
		return false, err
	}

	log.Debug("Ready to query for health checks")
	latestElResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "latest", 15)
	if err != nil {
		return false, err
	}
	latestElArtifact := e.convertResultToArtifact(e.prevHealthCheckResult.LatestElBlockResult, latestElResult)

	finalElResult, err := e.getExecBlockConsensus(ctx, execRpcClients, "finalized", 3)
	if err != nil {
		return false, err
	}
	finalElArtifact := e.convertResultToArtifact(e.prevHealthCheckResult.FinalizedElBlockResult, finalElResult)

	log.Debugf("Finalization -> latest lag: %d", latestElResult.ConsensusBlock-finalElResult.ConsensusBlock)

	latestClResult, err := e.getBeaconClientConsensus(ctx, beaconRpcClients, "head", 15)
	if err != nil {
		return false, err
	}
	latestClArtifact := e.convertResultToArtifact(e.prevHealthCheckResult.LatestClBlockResult, latestClResult)

	finalClResult, err := e.getBeaconClientConsensus(ctx, beaconRpcClients, "finalized", 3)
	if err != nil {
		return false, err
	}
	finalClArtifact := e.convertResultToArtifact(e.prevHealthCheckResult.FinalizedClBlockResult, finalClResult)

	results := &healthCheckResult{
		LatestElBlockResult:    latestElArtifact,
		FinalizedElBlockResult: finalElArtifact,
		LatestClBlockResult:    latestClArtifact,
		FinalizedClBlockResult: finalClArtifact,
	}

	e.prevHealthCheckResult = results

	passed, err := e.AllChecksPassed(results)
	if err != nil {
		return false, err
	}
	if passed {
		return true, nil
	} else {
		return false, nil
	}
}

func (e *healthChecker) AllChecksPassed(checks interface{}) (bool, error) {
	checksCasted, ok := checks.(*healthCheckResult)
	if !ok {
		return false, stacktrace.NewError("unable to convert checks %s to healthCheckResult", checks)
	}
	return checksCasted.AllChecksPassed(), nil
}

func (e *healthChecker) PopFinalResult() interface{} {
	return e.prevHealthCheckResult
}

func (e *healthChecker) convertResultToArtifact(
	prevArtifact *healthTypes.BlockConsensusArtifact,
	result *healthTypes.BlockConsensusTestResult) *healthTypes.BlockConsensusArtifact {

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

	return &healthTypes.BlockConsensusArtifact{
		BlockConsensusTestResult:       result,
		DidUnfaultedNodesFail:          didUnfaultedNodesFail,
		DidUnfaultedNodesNeedToRecover: didUnfaultedNodesNeedToRecover,
		NodeRecoveryTimeSeconds:        recoveredClients,
	}
}

func (e *healthCheckResult) AllChecksPassed() bool {
	if len(e.LatestElBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(e.LatestElBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(e.FinalizedElBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(e.FinalizedElBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(e.LatestClBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(e.LatestClBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(e.FinalizedClBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(e.FinalizedClBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	return true
}
