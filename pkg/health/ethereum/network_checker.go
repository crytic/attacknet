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

	startKill := false
	for {
		log.Info("running health check")

		latestForkVotes, _, finalizedForkVotes, err := getExecNetworkStabilizedConsensus(ctx, rpcClients, 3)
		if latestForkVotes != nil {
			consensusBlockNum, _, _, _ := determineForkConsensus(latestForkVotes)
			log.Infof("Consensus latest head: %d", consensusBlockNum[0].BlockNumber)
		}

		if err != nil {
			if err == UnableToReachLatestConsensusError {
				consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(latestForkVotes)
				reportConsensusDataToLogger("latest", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
				startKill = true
				time.Sleep(time.Second * 1)
				continue
			}
			if err == UnableToReachFinalConsensusError {
				consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(finalizedForkVotes)
				reportConsensusDataToLogger("finalized", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
				startKill = true
				time.Sleep(time.Second * 1)
				continue
			}
			return nil, err
		}

		// now close clients

		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(finalizedForkVotes)
		reportConsensusDataToLogger("finalized", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
		finalizedHead := consensusBlockNum[0].BlockNumber

		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash = determineForkConsensus(latestForkVotes)
		reportConsensusDataToLogger("latest", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
		latestHead := consensusBlockNum[0].BlockNumber

		// return err?
		log.Infof("Finalization -> latest lag: %d", latestHead-finalizedHead)
		//log.Infof("Safe -> Final lag: %d", safeHead-finalizedHead)

		if len(wrongBlockNum) == 0 && startKill {
			for _, c := range rpcClients {
				c.Close()
			}
			break
		}

		if len(wrongBlockNum) > 0 {
			if !startKill {
				startKill = true
			}
		}
		time.Sleep(time.Second * 1)
	}

	return nil, nil
}
