package suite

import (
	"attacknet/cmd/pkg/types"
	"time"
)

const clockSkewGracePeriod = time.Second * 1800
const containerRestartGracePeriod = time.Second * 3600

func composeNodeClockSkewTest(description string, targets []*ChaosTargetSelector, skew, duration string) (*types.SuiteTest, error) {
	var steps []types.PlanStep
	s, err := composeNodeClockSkewPlanSteps(targets, skew, duration)
	if err != nil {
		return nil, err
	}
	steps = append(steps, s...)

	waitStep := composeWaitForFaultCompletionStep()
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

func composeNodeRestartTest(description string, targets []*ChaosTargetSelector) (*types.SuiteTest, error) {
	var steps []types.PlanStep

	s, err := composeNodeRestartSteps(targets)
	if err != nil {
		return nil, err
	}
	steps = append(steps, s...)

	waitStep := composeWaitForFaultCompletionStep()
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

//func buildCpuPressureTest(description string, targets []*ChaosTargetSelector, pressure int) (*types.SuiteTest, error) {
