package network

type ExecClientConf struct {
	Type        string
	Image       string
	ExtraLabels map[string]string
}

type ConsensusClientConf struct {
	Type                string
	Image               string
	HasValidatorSidecar bool
	ValidatorImage      string
	ExtraLabels         map[string]string
}

type Validator struct {
	Index     int
	Execution *ExecClientConf
	Consensus *ConsensusClientConf
}
