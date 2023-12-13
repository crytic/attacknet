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

	// For now, we will simply ignore the edge case where these queries occur on a block production boundary.
	// We might have to fix before release.

	/*
		finalizedForkVotes, err := getExecNetworkConsensus(ctx, rpcClients, "finalized")
		if err != nil {
			return nil, err
		}

		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(finalizedForkVotes)
		if len(wrongBlockNum) > 0 {
			log.Warn("Consensus issue with finalized fork votes. Waiting 5s and trying again.")
			time.Sleep(5 * time.Second)
			finalizedForkVotes, err = getExecNetworkConsensus(ctx, rpcClients, "finalized")
			if err != nil {
				return nil, err
			}
		}

		latestForkVotes, err := getExecNetworkConsensus(ctx, rpcClients, "latest")
		if err != nil {
			return nil, err
		}
		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash = determineForkConsensus(latestForkVotes)
		if len(wrongBlockNum) > 0 {
			log.Warn("Consensus issue with latest fork votes. Waiting 5s and trying again.")
			time.Sleep(5 * time.Second)
			latestForkVotes, err = getExecNetworkConsensus(ctx, rpcClients, "latest")
			if err != nil {
				return nil, err
			}
		}

		safeForkVotes, err := getExecNetworkConsensus(ctx, rpcClients, "safe")
		if err != nil {
			return nil, err
		}
		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash = determineForkConsensus(safeForkVotes)
		if len(wrongBlockNum) > 0 {
			log.Warn("Consensus issue with safe fork votes. Waiting 5s and trying again.")
			time.Sleep(5 * time.Second)
			safeForkVotes, err = getExecNetworkConsensus(ctx, rpcClients, "safe")
			if err != nil {
				return nil, err
			}
		}
	*/

	startKill := false
	for {
		log.Info("running health check")

		latestForkVotes, safeForkVotes, finalizedForkVotes, err := getExecNetworkStabilizedConsensus(ctx, rpcClients, 3)
		if latestForkVotes != nil {
			consensusBlockNum, _, _, _ := determineForkConsensus(latestForkVotes)
			log.Infof("Consensus latest head: %d", consensusBlockNum[0].BlockNumber)
		}

		if err != nil {
			if err == UnableToReachLatestConsensusError {
				consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(latestForkVotes)
				reportConsensusDataToLogger("latest", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
				return nil, nil
			}
			if err == UnableToReachSafeConsensusError {
				consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(safeForkVotes)
				reportConsensusDataToLogger("safe", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
				startKill = true
				time.Sleep(time.Second * 1)
				continue
				//return nil, nil
			}
			if err == UnableToReachFinalConsensusError {
				consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(finalizedForkVotes)
				reportConsensusDataToLogger("finalized", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
				return nil, nil
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

		consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash = determineForkConsensus(safeForkVotes)
		reportConsensusDataToLogger("safe", consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
		safeHead := consensusBlockNum[0].BlockNumber

		// return err?
		log.Infof("Finalization -> latest lag: %d", latestHead-finalizedHead)
		log.Infof("Safe -> Final lag: %d", safeHead-finalizedHead)

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
