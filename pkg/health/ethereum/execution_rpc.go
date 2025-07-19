package ethereum

import (
	"attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"fmt"
	geth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"slices"
	"time"
)

type ExecClientRPC struct {
	session *kubernetes.PortForwardsSession
	client  *ethclient.Client
}

func (e *EthNetworkChecker) getExecBlockConsensus(ctx context.Context, clients []*ExecClientRPC, blockType string, maxAttempts int) (*types.BlockConsensusTestResult, error) {
	forkChoice, err := getExecNetworkConsensus(ctx, clients, blockType)
	if err != nil {
		return nil, err
	}
	// determine whether the nodes are in consensus
	consensusBlockNum, wrongBlockNum, consensusBlockHash, wrongBlockHash := determineForkConsensus(forkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts > 0 {
			log.Debugf("Nodes not at consensus for %s block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", blockType, maxAttempts-1)
			time.Sleep(1 * time.Second)
			return e.getExecBlockConsensus(ctx, clients, blockType, maxAttempts-1)
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

func (e *EthNetworkChecker) dialToExecutionClients(ctx context.Context) ([]*ExecClientRPC, error) {
	labelKey := "kurtosistech.com.custom/ethereum-package.client-type"
	labelValue := "execution"
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

	log.Debugf("Starting port forward sessions to %d pods", len(podsToHealthCheck))
	portForwardSessions, err := e.kubeClient.StartMultiPortForwardToLabeledPods(
		podsToHealthCheck,
		labelKey,
		labelValue,
		8545)
	if err != nil {
		return nil, err
	}

	// dial out to clients
	rpcClients := make([]*ExecClientRPC, len(portForwardSessions))
	for i, s := range portForwardSessions {
		client, err := dialExecRpcClient(s)
		if err != nil {
			return nil, err
		}
		rpcClients[i] = client
	}
	return rpcClients, nil
}

func dialExecRpcClient(session *kubernetes.PortForwardsSession) (*ExecClientRPC, error) {
	c, err := ethclient.Dial(fmt.Sprintf("http://localhost:%d", session.LocalPort))
	if err != nil {
		return nil, stacktrace.Propagate(err, "err while dialing RPC for %s", session.Pod.GetName())
	}
	return &ExecClientRPC{session: session, client: c}, nil
}

func (c *ExecClientRPC) Close() {
	c.client.Close()
	c.session.Close()
}

func (c *ExecClientRPC) GetLatestBlockBy(ctx context.Context, blockType string) (*ClientForkChoice, error) {
	// todo: handle pods that died and we didn't expect it
	var head *geth.Header
	var choice *ClientForkChoice
	err := c.client.Client().CallContext(ctx, &head, "eth_getBlockByNumber", blockType, false)
	if err != nil {
		notFinalizingErrors := []string{
			"safe block not found",      //geth
			"finalized block not found", //geth
			"Unknown block",             //erigon
			"Unknown block error",       //nethermind
			"unknown block",             //reth
		}

		noFinalBlockFound := slices.Contains(notFinalizingErrors, err.Error())

		if noFinalBlockFound {
			choice = &ClientForkChoice{
				Pod:         c.session.Pod,
				BlockNumber: 0,
				BlockHash:   "None",
			}
		} else {
			return nil, stacktrace.Propagate(err, "error while calling RPC for client %s", c.session.Pod.GetName())
		}
	} else {
		blockNum := head.Number.Uint64()
		hash := head.Hash().String()
		if blockNum == 0 {
			// use none for hash
			hash = "None"
		}
		choice = &ClientForkChoice{
			BlockNumber: blockNum,
			BlockHash:   hash,
			Pod:         c.session.Pod,
		}
	}
	return choice, nil
}
