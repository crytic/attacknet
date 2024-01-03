package network

func createGethClient() *ExecClientConf {
	return &ExecClientConf{Type: "geth", Image: "ethereum/client-go:latest", ExtraLabels: make(map[string]string)}
}

func createRethClient() *ExecClientConf {
	return &ExecClientConf{Type: "reth", Image: "ghcr.io/paradigmxyz/reth:v0.1.0-alpha.13", ExtraLabels: make(map[string]string)}
}

func createNethermindClient() *ExecClientConf {
	return &ExecClientConf{Type: "nethermind", Image: "nethermind/nethermind:1.23.0", ExtraLabels: make(map[string]string)}
}

func createErigonClient() *ExecClientConf {
	return &ExecClientConf{Type: "erigon", Image: "thorax/erigon:v2.53.4", ExtraLabels: make(map[string]string)}
}

func createBesuClient() *ExecClientConf {
	return &ExecClientConf{Type: "besu", Image: "hyperledger/besu:latest", ExtraLabels: make(map[string]string)}
}
