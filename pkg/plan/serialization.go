package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"strings"
)

func SerializeNetworkTopology(nodes []*network.Node, config *network.GenesisConfig) ([]byte, error) {
	serializableNodes := serializeNodes(nodes)

	netConfig := &EthKurtosisConfig{
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

func DeserializeNetworkTopology(conf []byte) ([]*network.Node, error) {
	parsedConf := EthKurtosisConfig{}
	err := yaml.Unmarshal(conf, &parsedConf)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to parse eth network types")
	}

	var nodes []*network.Node

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

		node := &network.Node{
			Index:          i + 1,
			ConsensusVotes: votesPerNode,
			Consensus: &network.ConsensusClient{
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
			Execution: &network.ExecutionClient{
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

func serializeNodes(nodes []*network.Node) []*Participant {
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
