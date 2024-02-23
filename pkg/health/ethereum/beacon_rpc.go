package ethereum

import (
	"attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type BeaconClientRpc struct {
	session *kubernetes.PortForwardsSession
	client  eth2client.BeaconBlockHeadersProvider
}

func (e *EthNetworkChecker) getBeaconClientConsensus(ctx context.Context, clients []*BeaconClientRpc, blockType string, maxAttempts int) (*types.BlockConsensusTestResult, error) {
	forkChoice, err := getBeaconNetworkConsensus(ctx, clients, blockType)
	if err != nil {
		return nil, err
	}
	// determine whether the nodes are in consensus
	consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(forkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts > 0 {
			log.Debugf("Nodes not at consensus for %s block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", blockType, maxAttempts-1)
			time.Sleep(1 * time.Second)
			return e.getBeaconClientConsensus(ctx, clients, blockType, maxAttempts-1)
		} else {
			reportConsensusDataToLogger(blockType, consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
		}
	}

	blockNumWrong := make(map[string]uint64)
	for _, node := range wrongBlockNum {
		blockNumWrong[node.Pod.GetName()] = node.BlockNumber
	}

	blockHashWrong := make(map[string]string)

	for _, node := range wrongBlockHash {
		blockHashWrong[node.Pod.GetName()] = node.BlockHash
	}
	reportConsensusDataToLogger(blockType, consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash)
	return &types.BlockConsensusTestResult{
		ConsensusBlock:              (consensusBlockNum)[0].BlockNumber,
		ConsensusHash:               consensusBlockHash[0].BlockHash,
		FailingClientsReportedBlock: blockNumWrong,
		FailingClientsReportedHash:  blockHashWrong,
	}, nil
}

func (e *EthNetworkChecker) dialToBeaconClients(ctx context.Context) ([]*BeaconClientRpc, error) {
	labelKey := "kurtosistech.com.custom/ethereum-package.client-type"
	labelValue := "beacon"
	podsToHealthCheck, err := getPodsToHealthCheck(
		ctx,
		e.kubeClient,
		e.podsUnderTest,
		e.podsUnderTestLookup,
		labelKey,
		labelValue)
	if err != nil {
		return nil, err
	}

	// todo: fix this when kurtosis pkg supports setting the port
	var port4000Batch []kubernetes.KubePod
	var port3500Batch []kubernetes.KubePod

	for _, pod := range podsToHealthCheck {
		if strings.Contains(pod.GetName(), "prysm") {
			port3500Batch = append(port3500Batch, pod)
		} else {
			port4000Batch = append(port4000Batch, pod)
		}
	}

	log.Debugf("Starting port forward sessions to %d pods", len(podsToHealthCheck))

	portForwardSessions3500, err := e.kubeClient.StartMultiPortForwardToLabeledPods(
		port3500Batch,
		labelKey,
		labelValue,
		3500)
	if err != nil {
		return nil, err
	}

	portForwardSessions4000, err := e.kubeClient.StartMultiPortForwardToLabeledPods(
		port4000Batch,
		labelKey,
		labelValue,
		4000)
	if err != nil {
		return nil, err
	}

	portForwardSessions := append(portForwardSessions3500, portForwardSessions4000...)

	// dial out to clients
	rpcClients := make([]*BeaconClientRpc, len(portForwardSessions))
	for i, s := range portForwardSessions {
		client, err := dialBeaconRpcClient(ctx, s)
		if err != nil {
			return nil, err
		}
		rpcClients[i] = client
	}
	return rpcClients, nil
}

func dialBeaconRpcClient(ctx context.Context, session *kubernetes.PortForwardsSession) (*BeaconClientRpc, error) {
	// 3 attempts
	retryCount := 8
	for i := 0; i <= retryCount; i++ {
		httpClient, err := http.New(ctx,
			http.WithAddress(fmt.Sprintf("http://localhost:%d", session.LocalPort)),
			http.WithLogLevel(zerolog.WarnLevel),
		)
		if err != nil {
			if i == retryCount {
				return nil, stacktrace.Propagate(err, "err while dialing RPC for %s", session.Pod.GetName())
			} else {
				time.Sleep(1 * time.Second)
				continue
			}

		}
		provider, isProvider := httpClient.(eth2client.BeaconBlockHeadersProvider)
		if !isProvider {
			return nil, stacktrace.NewError("unable to cast http client to beacon rpc provider for %s", session.Pod.GetName())
		}
		return &BeaconClientRpc{
			session: session,
			client:  provider,
		}, nil
	}
	return nil, stacktrace.NewError("unreachable beacon rpc")
}

func (c *BeaconClientRpc) Close() {
	c.session.Close()
}

func (c *BeaconClientRpc) GetLatestBlockBy(ctx context.Context, blockType string) (*ClientForkChoice, error) {
	// todo: handle pods that died and we didn't expect it
	result, err := c.client.BeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{Block: blockType})
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 404:
				if blockType == "finalized" {
					choice := &ClientForkChoice{
						Pod:         c.session.Pod,
						BlockNumber: 0,
						BlockHash:   "None",
					}
					return choice, nil
				}
			}
		}
		// chock it up to a failure we need to retry
		// note: at this time this retry logic isn't actually hooked up. I havent seen any failures to hit this RPC
		// endpoint yet, so setting up a retry mechanism may just be over-engineering.
		choice := &ClientForkChoice{
			Pod:         c.session.Pod,
			BlockNumber: 0,
			BlockHash:   "N/A",
		}
		return choice, nil
		//return nil, stacktrace.Propagate(err, "Unable to query for blockType %s with client for %s", blockType, c.session.Pod.GetName())
	}

	slot := uint64(result.Data.Header.Message.Slot)
	bodyHash := hex.EncodeToString(result.Data.Header.Message.BodyRoot[:])

	if slot == 0 && blockType == "finalized" {
		return &ClientForkChoice{
			Pod:         c.session.Pod,
			BlockNumber: slot,
			BlockHash:   "None",
		}, nil
	} else {
		return &ClientForkChoice{
			Pod:         c.session.Pod,
			BlockNumber: slot,
			BlockHash:   bodyHash,
		}, nil
	}
}
