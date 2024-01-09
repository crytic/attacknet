package network

import "attacknet/cmd/pkg/plan/types"

func createGethClient() *types.ExecutionClient {
	return &types.ExecutionClient{
		Type:           "geth",
		Image:          "ethereum/client-go:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createRethClient() *types.ExecutionClient {
	return &types.ExecutionClient{
		Type:           "reth",
		Image:          "ghcr.io/paradigmxyz/reth:v0.1.0-alpha.13",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createNethermindClient() *types.ExecutionClient {
	return &types.ExecutionClient{
		Type:           "nethermind",
		Image:          "nethermind/nethermind:1.23.0",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createErigonClient() *types.ExecutionClient {
	return &types.ExecutionClient{
		Type:           "erigon",
		Image:          "thorax/erigon:v2.53.4",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createBesuClient() *types.ExecutionClient {
	return &types.ExecutionClient{
		Type:           "besu",
		Image:          "hyperledger/besu:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}
