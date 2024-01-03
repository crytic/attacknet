package network

func createPrysmClient() *ConsensusClientConf {
	return &ConsensusClientConf{
		Type:                "prysm",
		Image:               "prysmaticlabs/prysm-beacon-chain:latest",
		ValidatorImage:      "prysmaticlabs/prysm-validator:latest",
		HasValidatorSidecar: true,
		ExtraLabels:         make(map[string]string)}
}

func createLighthouseClient() *ConsensusClientConf {
	return &ConsensusClientConf{
		Type:                "lighthouse",
		Image:               "sigp/lighthouse:latest",
		HasValidatorSidecar: true,
		ExtraLabels:         make(map[string]string)}
}

func createTekuClient() *ConsensusClientConf {
	return &ConsensusClientConf{
		Type:                "teku",
		Image:               "consensys/teku:23.12.0",
		HasValidatorSidecar: false,
		ExtraLabels:         make(map[string]string)}
}

func createLodestarClient() *ConsensusClientConf {
	return &ConsensusClientConf{
		Type:                "lodestar",
		Image:               "chainsafe/lodestar:v1.12.1",
		HasValidatorSidecar: true,
		ExtraLabels:         make(map[string]string)}
}

func createNimbusClient() *ConsensusClientConf {
	return &ConsensusClientConf{
		Type:                "nimbus",
		Image:               "statusim/nimbus-eth2:amd64-v23.11.0",
		HasValidatorSidecar: true,
		ExtraLabels:         make(map[string]string)}
}
