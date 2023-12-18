package types

type AttacknetConfig struct {
	GrafanaPodName             string `yaml:"grafanaPodName"`
	GrafanaPodPort             string `yaml:"grafanaPodPort"`
	AllowPostFaultInspection   bool   `yaml:"allowPostFaultInspection"`
	WaitBeforeInjectionSeconds uint32 `yaml:"waitBeforeInjectionSeconds"`
	ReuseDevnetBetweenRuns     bool   `yaml:"reuseDevnetBetweenRuns"`
	ExistingDevnetNamespace    string `yaml:"existingDevnetNamespace"`
}

type HarnessConfig struct {
	NetworkType       string `yaml:"networkType"`
	NetworkPackage    string `yaml:"networkPackage"`
	NetworkConfigPath string `yaml:"networkConfig"`
}

type HarnessConfigParsed struct {
	NetworkType    string
	NetworkPackage string
	NetworkConfig  []byte
}

type SuiteTestConfigs struct {
	Tests []SuiteTest `yaml:"tests"`
}

type SuiteTest struct {
	TestName  string     `yaml:"testName"`
	PlanSteps []PlanStep `yaml:"planSteps"`
}

type Config struct {
	AttacknetConfig AttacknetConfig  `yaml:"attacknetConfig"`
	HarnessConfig   HarnessConfig    `yaml:"harnessConfig"`
	TestConfig      SuiteTestConfigs `yaml:"testConfig"`
}

type ConfigParsed struct {
	AttacknetConfig AttacknetConfig
	HarnessConfig   HarnessConfigParsed
	TestConfig      SuiteTestConfigs
}

type StepType string

const (
	InvalidStepType        StepType = ""
	InjectFault            StepType = "injectFault"
	WaitForFaultCompletion StepType = "waitForFaultCompletion"
	WaitForDuration        StepType = "waitForDuration"
	WaitForHealthChecks    StepType = "waitForHealthChecks"
)

type PlanStep struct {
	StepType        StepType               `yaml:"stepType"`
	StepDescription string                 `yaml:"description"`
	Spec            map[string]interface{} `yaml:",inline"`
}
