package ethereum

import (
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"fmt"
	geth "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kurtosis-tech/stacktrace"
)

type ExecRpcClient struct {
	session *kubernetes.PortForwardsSession
	client  *ethclient.Client
}

func CreateExecRpcClient(session *kubernetes.PortForwardsSession) (*ExecRpcClient, error) {
	c, err := ethclient.Dial(fmt.Sprintf("http://localhost:%d", session.LocalPort))
	if err != nil {
		return nil, stacktrace.Propagate(err, "err while dialing RPC for %s", session.Pod.GetName())
	}
	return &ExecRpcClient{session: session, client: c}, nil
}

func (c *ExecRpcClient) Close() {
	c.client.Close()
	c.session.Close()
}

func (c *ExecRpcClient) GetLatestBlockBy(ctx context.Context, blockType string) (*ClientForkChoice, error) {
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

		noFinalBlockFound := false
		for _, msg := range notFinalizingErrors {
			if err.Error() == msg {
				noFinalBlockFound = true
				break
			}
		}

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
