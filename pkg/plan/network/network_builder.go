package network

import (
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

func serializeNodes(nodes []*Node) []*Participant {
	participants := make([]*Participant, len(nodes))
	for i, node := range nodes {
		consensusImage := node.Consensus.Image

		// prysm contingency
		if node.Consensus.HasValidatorSidecar && node.Consensus.ValidatorImage != "" {
			consensusImage = consensusImage + fmt.Sprintf(",%s", node.Consensus.ValidatorImage)
		}

		p := &Participant{
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

func createNodesForElTesting(index int, execClient ClientVersion, consensusClients map[string]ClientVersion) ([]*Node, error) {
	var nodes []*Node

	for _, consensusClient := range consensusClients {
		node := buildNode(index, execClient, consensusClient)
		nodes = append(nodes, node)

		index += 1
	}
	return nodes, nil
}

func createBootnode(execClients, consensusClients map[string]ClientVersion) (*Node, error) {
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

func loadClientConfig(clientConfigPath string) (executionClients, consensusClients map[string]ClientVersion, networkConfig *GenesisConfig, err error) {
	bs, err := os.ReadFile(clientConfigPath)
	if err != nil {
		return nil, nil, nil, stacktrace.Propagate(err, "could not read client config file on path %s", clientConfigPath)
	}

	var config PlanConfig
	err = yaml.Unmarshal(bs, &config)
	if err != nil {
		return nil, nil, nil, stacktrace.Propagate(err, "unable to unmarshal planner config in %s", clientConfigPath)
	}

	populateClientMap := func(li []ClientVersion) (map[string]ClientVersion, error) {
		clients := make(map[string]ClientVersion)
		for _, client := range li {
			_, exists := clients[client.Name]
			if exists {
				return nil, stacktrace.NewError("duplicate configuration for client %s", client.Name)
			}
			clients[client.Name] = client
		}
		return clients, nil
	}

	executionClients, err = populateClientMap(config.ExecutionClients)
	if err != nil {
		return nil, nil, nil, err
	}

	consensusClients, err = populateClientMap(config.ConsensusClients)
	if err != nil {
		return nil, nil, nil, err
	}

	return executionClients, consensusClients, &config.NetworkParams, nil
}

func BuildExecTesterNetwork(execClient string, clientConfigPath string) ([]*Node, *GenesisConfig, error) {
	execClients, consensusClients, networkConfig, err := loadClientConfig(clientConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// make sure execClient actually exists
	clientUnderTest, ok := execClients[execClient]
	if !ok {
		return nil, nil, stacktrace.NewError("unknown execution client %s", execClient)
	}

	var nodes []*Node
	index := 0
	bootnode, err := createBootnode(execClients, consensusClients)
	if err != nil {
		return nil, nil, err
	}
	nodes = append(nodes, bootnode)
	index += 1

	n, err := createNodesForElTesting(index, clientUnderTest, consensusClients)
	nodes = append(nodes, n...)
	if err != nil {
		return nil, nil, err
	}

	return nodes, networkConfig, nil
}

func SerializeNetworkConfig(nodes []*Node, config *GenesisConfig) ([]byte, error) {
	serializableNodes := serializeNodes(nodes)

	netConfig := &EthNetConfig{
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

func ParseKurtosisNetworkConfig(conf []byte) ([]*Node, error) {
	parsedConf := EthNetConfig{}
	err := yaml.Unmarshal(conf, &parsedConf)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to parse eth network config")
	}

	var nodes []*Node

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

		node := &Node{
			Index:          i + 1,
			ConsensusVotes: votesPerNode,
			Consensus: &ConsensusClient{
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
			Execution: &ExecutionClient{
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
