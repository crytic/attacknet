package network

import (
	"github.com/kurtosis-tech/stacktrace"
	"strings"
)

const defaultClCpu = 1000
const defaultValCpu = 500

const defaultClMem = 1536
const defaultValMem = 512

func composeConsensusTesterNetwork(nodeMultiplier int, consensusClient string, execClientList []ClientVersion, consClientMap map[string]ClientVersion) ([]*Node, error) {
	// make sure consensusClient actually exists
	clientUnderTest, ok := consClientMap[consensusClient]
	if !ok {
		return nil, stacktrace.NewError("unknown consensus client %s", consensusClient)
	}

	// start from 2 because bootnode is index 1
	index := 2
	nodes, err := composeNodesForClTesting(nodeMultiplier, index, clientUnderTest, execClientList)
	return nodes, err
}

func composeNodesForClTesting(nodeMultiplier, index int, consensusClient ClientVersion, execClients []ClientVersion) ([]*Node, error) {
	var nodes []*Node

	for _, execClient := range execClients {
		for i := 0; i < nodeMultiplier; i++ {
			node := buildNode(index, execClient, consensusClient)
			nodes = append(nodes, node)

			index += 1
		}
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
