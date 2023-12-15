package types

import "time"

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

type PlanStepType string

const (
	InjectFault            PlanStepType = "RunSingleFault"
	WaitForFaultCompletion PlanStepType = "WaitForFaultCompletion"
	WaitForDuration        PlanStepType = "WaitForDuration"
	WaitForHealthChecks    PlanStepType = "WaitForHealthChecks"
)

type TestPlanStep struct {

	// StepType
	//
}

type PlanStepSingleFault struct {
	StepDescription string                 `yaml:"description"`
	FaultSpec       map[string]interface{} `yaml:"chaosFaultSpec"`
}

type PlanStepWait struct {
	StepDescription string        `yaml:"description"`
	WaitAmount      time.Duration `yaml:"duration"`
}

type SuiteTestConfigs struct {
	Tests []SuiteTest `yaml:"tests"`
}

type SuiteTest struct {
	TestName  string        `yaml:"testName"`
	PlanSteps []interface{} `yaml:"planSteps"`
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
