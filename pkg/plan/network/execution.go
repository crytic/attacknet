package network

import (
	"github.com/kurtosis-tech/stacktrace"
)

const defaultElCpu = 1000
const defaultElMem = 1024

func composeExecTesterNetwork(nodeMultiplier int, execClient string, execClientMap, consClientMap map[string]ClientVersion) ([]*Node, error) {

	// make sure execClient actually exists
	clientUnderTest, ok := execClientMap[execClient]
	if !ok {
		return nil, stacktrace.NewError("unknown execution client %s", execClient)
	}

	// start from 2 because bootnode is index 1
	index := 2
	nodes, err := composeNodesForElTesting(nodeMultiplier, index, clientUnderTest, consClientMap)
	return nodes, err
}

func composeNodesForElTesting(nodeMultiplier, index int, execClient ClientVersion, consensusClients map[string]ClientVersion) ([]*Node, error) {
	var nodes []*Node

	for _, consensusClient := range consensusClients {
		for i := 0; i < nodeMultiplier; i++ {
			node := buildNode(index, execClient, consensusClient)
			nodes = append(nodes, node)

			index += 1
		}
	}
	return nodes, nil
}

func composeExecutionClient(config ClientVersion) *ExecutionClient {
	return &ExecutionClient{
		Type:           config.Name,
		Image:          config.Image,
		ExtraLabels:    make(map[string]string),
		CpuRequired:    defaultElCpu,
		MemoryRequired: defaultElMem,
	}
}
