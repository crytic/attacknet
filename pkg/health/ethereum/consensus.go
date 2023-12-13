package ethereum

import (
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

var UnableToReachLatestConsensusError = fmt.Errorf("there are nodes that disagree on the latest block")
var UnableToReachSafeConsensusError = fmt.Errorf("there are nodes that disagree on the safe block")
var UnableToReachFinalConsensusError = fmt.Errorf("there are nodes that disagree on the finalized block")

type ClientForkChoice struct {
	Pod         kubernetes.KubePod
	BlockNumber uint64
	BlockHash   string
}

func getExecNetworkStabilizedConsensus(ctx context.Context, nodeClients []*ExecRpcClient, maxAttempts int) ([]*ClientForkChoice, []*ClientForkChoice, []*ClientForkChoice, error) {
	latestForkChoice, err := getExecNetworkConsensus(ctx, nodeClients, "latest")
	if err != nil {
		return nil, nil, nil, err
	}
	_, wrongBlockNum, _, _ := determineForkConsensus(latestForkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts == 0 {
			return latestForkChoice, nil, nil, UnableToReachLatestConsensusError
		} else {
			log.Infof("Nodes not at consensus for latest block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", maxAttempts-1)
			time.Sleep(3 * time.Second)
			return getExecNetworkStabilizedConsensus(ctx, nodeClients, maxAttempts-1)

		}
	}

	safeForkChoice, err := getExecNetworkConsensus(ctx, nodeClients, "safe")
	if err != nil {
		return nil, nil, nil, err
	}
	_, wrongBlockNum, _, _ = determineForkConsensus(safeForkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts == 0 {
			return latestForkChoice, safeForkChoice, nil, UnableToReachSafeConsensusError
		} else {
			log.Infof("Nodes not at consensus for safe block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", maxAttempts-1)
			time.Sleep(3 * time.Second)
			return getExecNetworkStabilizedConsensus(ctx, nodeClients, maxAttempts-1)
		}
	}

	finalizedForkChoice, err := getExecNetworkConsensus(ctx, nodeClients, "finalized")
	if err != nil {
		return nil, nil, nil, err
	}
	_, wrongBlockNum, _, _ = determineForkConsensus(finalizedForkChoice)
	if len(wrongBlockNum) > 0 {
		if maxAttempts == 0 {
			return latestForkChoice, safeForkChoice, finalizedForkChoice, UnableToReachFinalConsensusError
		} else {
			log.Infof("Nodes not at consensus for finalized block. Waiting and re-trying in case we're on block propagation boundary. Attempts left: %d", maxAttempts-1)
			time.Sleep(3 * time.Second)
			return getExecNetworkStabilizedConsensus(ctx, nodeClients, maxAttempts-1)
		}
	}
	return latestForkChoice, safeForkChoice, finalizedForkChoice, nil
}

func getExecNetworkConsensus(ctx context.Context, nodeClients []*ExecRpcClient, blockType string) ([]*ClientForkChoice, error) {
	clientForkVotes := make([]*ClientForkChoice, len(nodeClients))
	for i, client := range nodeClients {
		choice, err := client.GetLatestBlockBy(ctx, blockType)
		if err != nil {
			return nil, err
		}

		clientForkVotes[i] = choice
	}
	return clientForkVotes, nil
}

func determineForkConsensus(nodes []*ClientForkChoice) (
	consensusBlockNum []*ClientForkChoice,
	wrongBlockNum []*ClientForkChoice,
	consensusBlockHash []*ClientForkChoice,
	wrongBlockHash []*ClientForkChoice) {

	// convert node votes to map
	blockVotes := make(map[uint64][]*ClientForkChoice)
	for _, vote := range nodes {
		blockVotes[vote.BlockNumber] = append(blockVotes[vote.BlockNumber], vote)
	}

	//var consensusBlock uint64
	var consensusBlockVotes int

	// determine consensus block height
	for _, v := range blockVotes {
		if len(v) > consensusBlockVotes {
			if consensusBlockVotes != 0 {
				wrongBlockNum = append(wrongBlockNum, consensusBlockNum...)
			}
			//consensusBlock = k
			consensusBlockVotes = len(v)
			consensusBlockNum = v
		} else {
			wrongBlockNum = append(wrongBlockNum, v...)
		}
	}

	// for the consensus block height, determine the consensus hash
	var hashVotes = make(map[string][]*ClientForkChoice)
	for _, vote := range consensusBlockNum {
		hashVotes[vote.BlockHash] = append(hashVotes[vote.BlockHash], vote)
	}
	//var consensusHash string
	var consensusHashVotes int
	for _, v := range hashVotes {
		if len(v) > consensusHashVotes {
			if consensusBlockVotes != 0 {
				wrongBlockHash = append(wrongBlockHash, consensusBlockHash...)
			}
			//consensusHash = k
			consensusHashVotes = len(v)
			consensusBlockHash = v
		} else {
			wrongBlockHash = append(wrongBlockHash, v...)
		}
	}
	return
}

func reportConsensusDataToLogger(consensusType string,
	consensusBlockNum []*ClientForkChoice,
	wrongBlockNum []*ClientForkChoice,
	consensusBlockHash []*ClientForkChoice,
	wrongBlockHash []*ClientForkChoice) {

	log.Infof("Consensus %s block height: %d", consensusType, consensusBlockNum[0].BlockNumber)
	if len(wrongBlockNum) > 0 {
		log.Warnf("Some nodes are out of consensus for %s", consensusType)
		for _, n := range wrongBlockNum {
			log.Warnf("---> Node: %s %s BlockHeight: %d BlockHash: %s", n.Pod.GetName(), consensusType, n.BlockNumber, n.BlockHash)
		}
	}

	log.Infof("Consensus %s block hash: %s", consensusType, consensusBlockHash[0].BlockHash)
	if len(wrongBlockHash) > 0 {
		log.Warnf("Some nodes are at the correct height, but with the wrong %s block hash", consensusType)
		for _, n := range wrongBlockHash {
			log.Warnf("---> Node: %s %s BlockHeight: %d BlockHash: %s", n.Pod.GetName(), consensusType, n.BlockNumber, n.BlockHash)
		}
	}
}
