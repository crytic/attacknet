package network

var execClients = map[string]func() *ExecClient{
	"geth":       createGethClient,
	"reth":       createRethClient,
	"nethermind": createNethermindClient,
	"erigon":     createErigonClient,
	"besu":       createBesuClient,
}
var _ = execClients

func createGethClient() *ExecClient {
	return &ExecClient{
		Type:           "geth",
		Image:          "ethereum/client-go:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createRethClient() *ExecClient {
	return &ExecClient{
		Type:           "reth",
		Image:          "ghcr.io/paradigmxyz/reth:v0.1.0-alpha.13",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createNethermindClient() *ExecClient {
	return &ExecClient{
		Type:           "nethermind",
		Image:          "nethermind/nethermind:1.23.0",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createErigonClient() *ExecClient {
	return &ExecClient{
		Type:           "erigon",
		Image:          "thorax/erigon:v2.53.4",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}

func createBesuClient() *ExecClient {
	return &ExecClient{
		Type:           "besu",
		Image:          "hyperledger/besu:latest",
		ExtraLabels:    make(map[string]string),
		CpuRequired:    1000,
		MemoryRequired: 1024,
	}
}
