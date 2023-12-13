package ethereum

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	log "github.com/sirupsen/logrus"
)
import "attacknet/cmd/pkg/health/types"

type EthNetworkChecker struct {
	kubeClient          *kubernetes.KubeClient
	podsUnderTest       []*chaos_mesh.PodUnderTest
	podsUnderTestLookup map[string]*chaos_mesh.PodUnderTest
}

func CreateEthNetworkChecker(kubeClient *kubernetes.KubeClient, podsUnderTest []*chaos_mesh.PodUnderTest) *EthNetworkChecker {
	// convert podsUnderTest to a lookup
	podsUnderTestMap := make(map[string]*chaos_mesh.PodUnderTest)

	for _, pod := range podsUnderTest {
		podsUnderTestMap[pod.Name] = pod
	}

	return &EthNetworkChecker{
		podsUnderTest:       podsUnderTest,
		podsUnderTestLookup: podsUnderTestMap,
		kubeClient:          kubeClient,
	}
}

func (e *EthNetworkChecker) RunAllChecks(ctx context.Context) ([]*types.CheckResult, error) {
	labelKey := "kurtosistech.com.custom/ethereum-package.client-type"
	labelValue := "execution"

	var podsToHealthCheck []kubernetes.KubePod
	// add pods under test that match the label criteria
	for _, pod := range e.podsUnderTest {
		if pod.MatchesLabel(labelKey, labelValue) && !pod.ExpectDeath {
			podsToHealthCheck = append(podsToHealthCheck, pod)
		}
	}
	// add pods that were not targeted by a fault
	bystanders, err := e.kubeClient.PodsMatchingLabel(ctx, labelKey, labelValue)
	if err != nil {
		return nil, err
	}
	for _, pod := range bystanders {
		_, match := e.podsUnderTestLookup[pod.GetName()]
		// don't add pods we've already added
		if !match {
			podsToHealthCheck = append(podsToHealthCheck, pod)
		}
	}

	log.Infof("Starting port forward sessions to %d pods", len(podsToHealthCheck))
	portForwardSessions, err := e.kubeClient.StartMultiPortForwardToLabeledPods(
		podsToHealthCheck,
		labelKey,
		labelValue,
		8545)
	if err != nil {
		return nil, err
	}

	// dial out to clients
	rpcClients := make([]*ExecRpcClient, len(portForwardSessions))
	for i, s := range portForwardSessions {
		client, err := CreateExecRpcClient(s)
		if err != nil {
			return nil, err
		}
		rpcClients[i] = client
	}

	log.Info("Ready to query for health checks")

	// For now, we will simply ignore the edge case where these queries occur on a block production boundary.
	// We might have to fix before release.
	clientForkVotes := make([]*ClientForkChoice, len(rpcClients))
	for i, client := range rpcClients {
		choice, err := client.GetLatestBlockBy(ctx, "finalized")
		if err != nil {
			return nil, err
		}

		clientForkVotes[i] = choice
	}

	// do other rpc things

	// now close clients
	for _, c := range rpcClients {
		c.Close()
	}

	consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(clientForkVotes)

	log.Infof("Consensus finalized block height: %d", consensusBlockNum[0].BlockNumber)
	if len(wrongBlockNum) > 0 {
		log.Warn("Some nodes are out of consensus.")
		for _, n := range wrongBlockNum {
			log.Warnf("---> Node: %s Finalized BlockHeight: %d BlockHash: %s", n.Pod.GetName(), n.BlockNumber, n.BlockHash)
		}
	}

	log.Infof("Consensus finalized block hash: %s", consensusBlockHash[0].BlockHash)
	if len(wrongBlockHash) > 0 {
		log.Warn("Some nodes are at the correct height, but with the wrong finalized block hash")
		for _, n := range wrongBlockHash {
			log.Warnf("---> Node: %s Finalized BlockHeight: %d BlockHash: %s", n.Pod.GetName(), n.BlockNumber, n.BlockHash)
		}
	}

	// return err?

	return nil, nil
}
