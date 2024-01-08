package network

import "strings"

const default_el_cpu = 1000
const default_cl_cpu = 1000
const default_val_cpu = 1000

const default_el_mem = 1024
const default_cl_mem = 2048
const default_val_mem = 1024

func buildConsensusClient(config ClientVersion) *ConsensusClient {
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
			CpuRequired:           default_cl_cpu,
			MemoryRequired:        default_cl_mem,
			SidecarCpuRequired:    default_val_cpu,
			SidecarMemoryRequired: default_val_mem,
		}
	} else {
		return &ConsensusClient{
			Type:                  config.Name,
			Image:                 image,
			HasValidatorSidecar:   false,
			ValidatorImage:        validatorImage,
			ExtraLabels:           make(map[string]string),
			CpuRequired:           default_cl_cpu,
			MemoryRequired:        default_cl_mem,
			SidecarCpuRequired:    0,
			SidecarMemoryRequired: 0,
		}
	}
}

func buildExecutionClient(config ClientVersion) *ExecutionClient {
	return &ExecutionClient{
		Type:           config.Name,
		Image:          config.Image,
		ExtraLabels:    make(map[string]string),
		CpuRequired:    default_cl_cpu,
		MemoryRequired: default_cl_mem,
	}
}

func buildNode(index int, execConf, consensusConf ClientVersion) *Node {
	return &Node{
		Index:     index,
		Execution: buildExecutionClient(execConf),
		Consensus: buildConsensusClient(consensusConf),
	}
}
