package network

import (
	"attacknet/cmd/pkg/plan/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"strings"
)

func serializeNodes(nodes []*types.Node) []*types.Participant {
	participants := make([]*types.Participant, len(nodes))
	for i, node := range nodes {
		consensusImage := node.Consensus.Image

		// prysm contingency
		if node.Consensus.HasValidatorSidecar && node.Consensus.ValidatorImage != "" {
			consensusImage = consensusImage + fmt.Sprintf(",%s", node.Consensus.ValidatorImage)
		}

		p := &types.Participant{
			ElClientType:  node.Execution.Type,
			ElClientImage: node.Execution.Image,

			ClClientType:  node.Consensus.Type,
			ClClientImage: consensusImage,

			ElMinCpu:    node.Execution.CpuRequired,
			ElMaxCpu:    node.Execution.CpuRequired,
			ElMinMemory: node.Execution.MemoryRequired,
			ElMaxMemory: node.Execution.MemoryRequired,

			ClMinCpu:    node.Consensus.CpuRequired,
			ClMaxCpu:    node.Consensus.CpuRequired,
			ClMinMemory: node.Consensus.MemoryRequired,
			ClMaxMemory: node.Consensus.MemoryRequired,

			ValMinCpu:    node.Consensus.SidecarCpuRequired,
			ValMaxCpu:    node.Consensus.SidecarCpuRequired,
			ValMinMemory: node.Consensus.SidecarMemoryRequired,
			ValMaxMemory: node.Consensus.SidecarMemoryRequired,
			Count:        1,
		}
		participants[i] = p
	}

	return participants
}

func createNodesForElTesting(index int, execClient types.ClientVersion, consensusClients map[string]types.ClientVersion) ([]*types.Node, error) {
	var nodes []*types.Node

	for _, consensusClient := range consensusClients {
		node := buildNode(index, execClient, consensusClient)
		nodes = append(nodes, node)

		index += 1
	}
	return nodes, nil
}

func createBootnode(execClients, consensusClients map[string]types.ClientVersion) (*types.Node, error) {
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

func clientListsToMaps(execClients, consClients []types.ClientVersion) (execClientMap, consClientMap map[string]types.ClientVersion, err error) {
	populateClientMap := func(li []types.ClientVersion) (map[string]types.ClientVersion, error) {
		clients := make(map[string]types.ClientVersion)
		for _, client := range li {
			_, exists := clients[client.Name]
			if exists {
				return nil, stacktrace.NewError("duplicate configuration for client %s", client.Name)
			}
			clients[client.Name] = client
		}
		return clients, nil
	}

	execClientMap, err = populateClientMap(execClients)
	if err != nil {
		return nil, nil, err
	}

	consClientMap, err = populateClientMap(consClients)
	if err != nil {
		return nil, nil, err
	}

	return execClientMap, consClientMap, nil
}

func ComposeNetworkTopology(client string, execClients, consClients []types.ClientVersion) ([]*types.Node, error) {
	if client == "all" {
		return nil, stacktrace.NewError("target client 'all' not supported yet")
	}

	isExecutionClient := false
	for _, execClient := range execClients {
		if execClient.Name == client {
			isExecutionClient = true
			break
		}
	}
	// assume already checked client is a member of consClients or execClients

	if isExecutionClient {
		return composeExecTesterNetwork(client, execClients, consClients)
	} else {
		// todo
		return nil, nil
	}
}

func composeExecTesterNetwork(execClient string, execClients, consClients []types.ClientVersion) ([]*types.Node, error) {
	execClientMap, consClientMap, err := clientListsToMaps(execClients, consClients)
	if err != nil {
		return nil, err
	}

	// make sure execClient actually exists
	clientUnderTest, ok := execClientMap[execClient]
	if !ok {
		return nil, stacktrace.NewError("unknown execution client %s", execClient)
	}

	var nodes []*types.Node
	index := 0
	bootnode, err := createBootnode(execClientMap, consClientMap)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bootnode)
	index += 1

	n, err := createNodesForElTesting(index, clientUnderTest, consClientMap)
	nodes = append(nodes, n...)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func SerializeNetworkTopology(nodes []*types.Node, config *types.GenesisConfig) ([]byte, error) {
	serializableNodes := serializeNodes(nodes)

	netConfig := &types.EthNetConfig{
		Participants: serializableNodes,
		NetParams:    *config,
		AdditionalServices: []string{
			"prometheus_grafana",
			"dora",
		},
		ParallelKeystoreGen: false,
	}

	bs, err := yaml.Marshal(netConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "intermediate yaml marshalling failed")
	}

	return bs, nil
}

func ParseKurtosisNetworkConfig(conf []byte) ([]*types.Node, error) {
	parsedConf := types.EthNetConfig{}
	err := yaml.Unmarshal(conf, &parsedConf)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to parse eth network types")
	}

	var nodes []*types.Node

	for i, participant := range parsedConf.Participants {
		hasSidecar := false
		consensusImage := participant.ClClientImage
		validatorImage := ""
		if participant.ValMaxCpu != 0 {
			hasSidecar = true
			if strings.Contains(consensusImage, ",") {
				images := strings.Split(consensusImage, ",")
				consensusImage = images[0]
				validatorImage = images[1]
			}
		}

		votesPerNode := parsedConf.NetParams.NumValKeysPerNode

		node := &types.Node{
			Index:          i + 1,
			ConsensusVotes: votesPerNode,
			Consensus: &types.ConsensusClient{
				Type:                  participant.ClClientType,
				Image:                 consensusImage,
				ValidatorImage:        validatorImage,
				HasValidatorSidecar:   hasSidecar,
				ExtraLabels:           map[string]string{},
				CpuRequired:           participant.ClMinCpu,
				MemoryRequired:        participant.ClMinMemory,
				SidecarCpuRequired:    participant.ValMinCpu,
				SidecarMemoryRequired: participant.ValMinMemory,
			},
			Execution: &types.ExecutionClient{
				Type:           participant.ElClientType,
				Image:          participant.ElClientImage,
				ExtraLabels:    map[string]string{},
				CpuRequired:    participant.ElMinCpu,
				MemoryRequired: participant.ElMinMemory,
			},
		}

		nodes = append(nodes, node)

	}
	return nodes, nil
}
