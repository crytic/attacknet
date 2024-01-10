package suite

import (
	"attacknet/cmd/pkg/types"
	"time"
)

const clockSkewGracePeriod = time.Second * 1800
const containerRestartGracePeriod = time.Second * 3600

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
			GracePeriod:  clockSkewGracePeriod,
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
			GracePeriod:  containerRestartGracePeriod,
		},
	}

	return test, nil
}

//func buildCpuPressureTest(description string, targets []*TargetSelector, pressure int) (*types.SuiteTest, error) {
