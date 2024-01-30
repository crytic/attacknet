package network

import (
	"github.com/kurtosis-tech/stacktrace"
)

const defaultElCpu = 1000
const defaultElMem = 1024

func composeExecTesterNetwork(bootEl, bootCl, execClient string, execClients, consClients []ClientVersion) ([]*Node, error) {
	execClientMap, consClientMap, err := clientListsToMaps(execClients, consClients)
	if err != nil {
		return nil, err
	}

	// make sure execClient actually exists
	clientUnderTest, ok := execClientMap[execClient]
	if !ok {
		return nil, stacktrace.NewError("unknown execution client %s", execClient)
	}

	var nodes []*Node
	index := 1
	bootnode, err := composeBootnode(bootEl, bootCl, execClientMap, consClientMap)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bootnode)
	index += 1

	n, err := composeNodesForElTesting(index, clientUnderTest, consClientMap)
	nodes = append(nodes, n...)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func composeNodesForElTesting(index int, execClient ClientVersion, consensusClients map[string]ClientVersion) ([]*Node, error) {
	var nodes []*Node

	for _, consensusClient := range consensusClients {
		node := buildNode(index, execClient, consensusClient)
		nodes = append(nodes, node)

		index += 1
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
