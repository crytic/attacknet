package suite

import (
	"attacknet/cmd/pkg/types"
	"time"
)

func buildNodeClockSkewTest(description string, targets []*TargetSelector, skew, duration string) (*types.SuiteTest, error) {
	var steps []types.PlanStep
	s, err := buildNodeClockSkewPlanSteps(targets, skew, duration)
	if err != nil {
		return nil, err
	}
	steps = append(steps, s...)

	waitStep := buildWaitForFaultCompletionStep()
	steps = append(steps, *waitStep)

	test := &types.SuiteTest{
		TestName:  description,
		PlanSteps: steps,
		HealthConfig: types.HealthCheckConfig{
			EnableChecks: true,
			GracePeriod:  time.Second * 120 * 2,
		},
	}

	return test, nil
}

func buildNodeRestartTest(description string, targets []*TargetSelector) (*types.SuiteTest, error) {
	var steps []types.PlanStep

	s, err := buildNodeRestartSteps(targets)
	if err != nil {
		return nil, err
	}
	steps = append(steps, s...)

	waitStep := buildWaitForFaultCompletionStep()
	steps = append(steps, *waitStep)

	test := &types.SuiteTest{
		TestName:  description,
		PlanSteps: steps,
		HealthConfig: types.HealthCheckConfig{
			EnableChecks: true,
			GracePeriod:  time.Second * 240 * 2,
		},
	}

	return test, nil
}

//func buildCpuPressureTest(description string, targets []*TargetSelector, pressure int) (*types.SuiteTest, error) {
