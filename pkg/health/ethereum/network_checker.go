package ethereum

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
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

	log.Debug("Ready to query for health checks")
	latestResult, err := e.getBlockConsensus(ctx, rpcClients, "latest", 3)
	if err != nil {
		return nil, err
	}
	finalResult, err := e.getBlockConsensus(ctx, rpcClients, "finalized", 3)
	if err != nil {
		return nil, err
	}

	log.Infof("Finalization -> latest lag: %d", latestResult.ConsensusBlockNum-finalResult.ConsensusBlockNum)

	// construct results
	results := make([]*types.CheckResult, 4)
	results[0] = latestResult.BlockNumResult
	results[1] = latestResult.BlockHashResult
	results[2] = finalResult.BlockNumResult
	results[3] = finalResult.BlockHashResult

	return results, nil
}

type getBlockConsensusResult struct {
	BlockNumResult     *types.CheckResult
	BlockHashResult    *types.CheckResult
	ConsensusBlockNum  uint64
	ConsensusBlockHash string
}

func (e *EthNetworkChecker) getBlockConsensus(ctx context.Context, clients []*ExecRpcClient, blockType string, maxAttempts int) (*getBlockConsensusResult, error) {
	forkChoice, err := getExecNetworkConsensus(ctx, clients, blockType)
	if err != nil {
		return nil, err
	}
	// determine whether the nodes are in consensus
	consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(forkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts > 0 {
			log.Debugf("Nodes not at consensus for %s block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", blockType, maxAttempts-1)
			time.Sleep(2 * time.Second)
			return e.getBlockConsensus(ctx, clients, blockType, maxAttempts-1)
		} else {
			reportConsensusDataToLogger(blockType, consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
		}
	}

	blockNumResult := &types.CheckResult{}
	blockNumResult.TestName = fmt.Sprintf("All nodes agree on %s block number", blockType)
	for _, node := range consensusBlockNum {
		blockNumResult.PodsPassing = append(blockNumResult.PodsPassing, node.Pod.GetName())
	}
	for _, node := range wrongBlockNum {
		blockNumResult.PodsFailing = append(blockNumResult.PodsFailing, node.Pod.GetName())
	}

	blockHashResult := &types.CheckResult{}
	blockHashResult.TestName = fmt.Sprintf("All nodes agree on %s block hash", blockType)
	for _, node := range consensusBlockHash {
		blockHashResult.PodsPassing = append(blockHashResult.PodsPassing, node.Pod.GetName())
	}
	for _, node := range wrongBlockHash {
		blockHashResult.PodsFailing = append(blockHashResult.PodsFailing, node.Pod.GetName())
	}
	reportConsensusDataToLogger(blockType, consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
	return &getBlockConsensusResult{
		blockNumResult,
		blockHashResult,
		consensusBlockNum[0].BlockNumber,
		consensusBlockHash[0].BlockHash,
	}, nil
}
