package ethereum

import "attacknet/cmd/pkg/kubernetes"

type ClientForkChoice struct {
	Pod         kubernetes.KubePod
	BlockNumber uint64
	BlockHash   string
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
