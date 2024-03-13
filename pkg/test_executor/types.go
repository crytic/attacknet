package test_executor

import (
	"time"
)

type PlanStepSingleFault struct {
	FaultSpec map[string]interface{} `yaml:"chaosFaultSpec"`
}

type PlanStepWaitForFaultCompletion struct {
}

type PlanStepWait struct {
	StepDescription string        `yaml:"description"`
	WaitAmount      time.Duration `yaml:"duration"`
}
