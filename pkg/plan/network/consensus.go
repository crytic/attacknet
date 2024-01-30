package network

import (
	"github.com/kurtosis-tech/stacktrace"
	"strings"
)

const defaultClCpu = 1000
const defaultValCpu = 1000

const defaultClMem = 2048
const defaultValMem = 1024

func composeConsensusTesterNetwork(bootEl, bootCl, consensusClient string, execClients, consClients []ClientVersion) ([]*Node, error) {
	execClientMap, consClientMap, err := clientListsToMaps(execClients, consClients)
	if err != nil {
		return nil, err
	}

	// make sure consensusClient actually exists
	clientUnderTest, ok := consClientMap[consensusClient]
	if !ok {
		return nil, stacktrace.NewError("unknown consensus client %s", consensusClient)
	}

	var nodes []*Node
	index := 1
	bootnode, err := composeBootnode(bootEl, bootCl, execClientMap, consClientMap)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bootnode)
	index += 1

	n, err := composeNodesForClTesting(index, clientUnderTest, execClientMap)
	nodes = append(nodes, n...)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func composeNodesForClTesting(index int, consensusClient ClientVersion, execClients map[string]ClientVersion) ([]*Node, error) {
	var nodes []*Node

	for _, execClient := range execClients {
		node := buildNode(index, execClient, consensusClient)
		nodes = append(nodes, node)

		index += 1
	}
	return nodes, nil
}

func composeConsensusClient(config ClientVersion) *ConsensusClient {
	image := config.Image
	validatorImage := ""

	if strings.Contains(config.Image, ",") {
		images := strings.Split(config.Image, ",")
		image = images[0]
		validatorImage = images[1]
	}
	if config.HasSidecar {
		return &ConsensusClient{
			Type:                  config.Name,
			Image:                 image,
			HasValidatorSidecar:   true,
			ValidatorImage:        validatorImage,
			ExtraLabels:           make(map[string]string),
			CpuRequired:           defaultClCpu,
			MemoryRequired:        defaultClMem,
			SidecarCpuRequired:    defaultValCpu,
			SidecarMemoryRequired: defaultValMem,
		}
	} else {
		return &ConsensusClient{
			Type:                  config.Name,
			Image:                 image,
			HasValidatorSidecar:   false,
			ValidatorImage:        validatorImage,
			ExtraLabels:           make(map[string]string),
			CpuRequired:           defaultClCpu,
			MemoryRequired:        defaultClMem,
			SidecarCpuRequired:    0,
			SidecarMemoryRequired: 0,
		}
	}
}
