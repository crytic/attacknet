package network

import "attacknet/cmd/pkg/plan/types"

func createPrysmClient() *types.ConsensusClient {
	return &types.ConsensusClient{
		Type:                  "prysm",
		Image:                 "prysmaticlabs/prysm-beacon-chain:latest",
		ValidatorImage:        "prysmaticlabs/prysm-validator:latest",
		HasValidatorSidecar:   true,
		ExtraLabels:           make(map[string]string),
		CpuRequired:           2000,
		MemoryRequired:        2048,
		SidecarCpuRequired:    1000,
		SidecarMemoryRequired: 1024,
	}
}

func createLighthouseClient() *types.ConsensusClient {
	return &types.ConsensusClient{
		Type:                  "lighthouse",
		Image:                 "sigp/lighthouse:latest",
		HasValidatorSidecar:   true,
		ExtraLabels:           make(map[string]string),
		CpuRequired:           2000,
		MemoryRequired:        2048,
		SidecarCpuRequired:    1000,
		SidecarMemoryRequired: 1024,
	}
}

func createTekuClient() *types.ConsensusClient {
	return &types.ConsensusClient{
		Type:                "teku",
		Image:               "consensys/teku:23.12.0",
		HasValidatorSidecar: false,
		ExtraLabels:         make(map[string]string),
		CpuRequired:         2000,
		MemoryRequired:      2048,
	}
}

func createLodestarClient() *types.ConsensusClient {
	return &types.ConsensusClient{
		Type:                  "lodestar",
		Image:                 "chainsafe/lodestar:v1.12.1",
		HasValidatorSidecar:   true,
		ExtraLabels:           make(map[string]string),
		CpuRequired:           2000,
		MemoryRequired:        2048,
		SidecarCpuRequired:    1000,
		SidecarMemoryRequired: 1024,
	}
}

func createNimbusClient() *types.ConsensusClient {
	return &types.ConsensusClient{
		Type:                  "nimbus",
		Image:                 "statusim/nimbus-eth2:amd64-v23.11.0",
		HasValidatorSidecar:   true,
		ExtraLabels:           make(map[string]string),
		CpuRequired:           2000,
		MemoryRequired:        2048,
		SidecarCpuRequired:    1000,
		SidecarMemoryRequired: 1024,
	}
}
