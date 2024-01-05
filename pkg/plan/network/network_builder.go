package network

import (
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
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

func buildNode(index int, execBuilder func() *ExecClient, consBuilder func() *ConsensusClient) *Node {
	return &Node{
		Index:     index,
		Execution: execBuilder(),
		Consensus: consBuilder(),
	}
}

func createNodesForElTesting(index int, execClient string) ([]*Node, error) {
	var nodes []*Node
	execBuilder, ok := execClients[execClient]
	if !ok {
		return nil, stacktrace.NewError("client %s does not have a builder", execClient)
	}

	for _, consBuilder := range consensusClients {
		node := buildNode(index, execBuilder, consBuilder)
		nodes = append(nodes, node)

		index += 1
	}
	return nodes, nil
}

func createBootnode() *Node {
	return buildNode(0, createGethClient, createLighthouseClient)
}

func BuildExecTesterNetwork(execClient string) ([]byte, error) {
	var nodes []*Node
	index := 0
	nodes = append(nodes, createBootnode())
	index += 1

	n, err := createNodesForElTesting(index, execClient)
	nodes = append(nodes, n...)
	if err != nil {
		return nil, err
	}

	serializableNodes := serializeNodes(nodes)

	netConfig := &EthNetConfig{
		Participants: serializableNodes,
		NetParams: &EthPackageNetworkParams{
			NumValKeysPerNode: 32,
		},
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

func ParseNetworkConfig(conf []byte) ([]*Node, error) {
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

		node := &Node{
			Index:          i + 1,
			ConsensusVotes: parsedConf.NetParams.NumValKeysPerNode,
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
			Execution: &ExecClient{
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
