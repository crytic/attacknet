package network

import (
	"github.com/kurtosis-tech/stacktrace"
)

func buildNode(index int, execConf, consensusConf ClientVersion) *Node {
	return &Node{
		Index:     index,
		Execution: composeExecutionClient(execConf),
		Consensus: composeConsensusClient(consensusConf),
	}
}

func composeBootnode(execClients, consensusClients map[string]ClientVersion) (*Node, error) {
	execConf, ok := execClients["geth"]
	if !ok {
		return nil, stacktrace.NewError("unable to load configuration for exec client geth")
	}
	consConf, ok := consensusClients["lighthouse"]
	if !ok {
		return nil, stacktrace.NewError("unable to load configuration for exec client lighthouse")
	}
	return buildNode(0, execConf, consConf), nil
}
