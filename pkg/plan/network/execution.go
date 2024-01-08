package network

func createGethClient() *ExecutionClient {
	return &ExecutionClient{
		Type:           "geth",
		Image:          "ethereum/client-go:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createRethClient() *ExecutionClient {
	return &ExecutionClient{
		Type:           "reth",
		Image:          "ghcr.io/paradigmxyz/reth:v0.1.0-alpha.13",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createNethermindClient() *ExecutionClient {
	return &ExecutionClient{
		Type:           "nethermind",
		Image:          "nethermind/nethermind:1.23.0",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createErigonClient() *ExecutionClient {
	return &ExecutionClient{
		Type:           "erigon",
		Image:          "thorax/erigon:v2.53.4",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createBesuClient() *ExecutionClient {
	return &ExecutionClient{
		Type:           "besu",
		Image:          "hyperledger/besu:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}
