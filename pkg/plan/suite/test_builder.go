package suite

import (
	"attacknet/cmd/pkg/types"
	"time"
)

func composeNodeClockSkewTest(description string, targets []*ChaosTargetSelector, skew, duration string, graceDuration *time.Duration) (*types.SuiteTest, error) {
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
			GracePeriod:  graceDuration,
		},
	}

	return test, nil
}

func composeNodeRestartTest(description string, targets []*ChaosTargetSelector, graceDuration *time.Duration) (*types.SuiteTest, error) {
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
			GracePeriod:  graceDuration,
		},
	}

	return test, nil
}

func composeIOLatencyTest(description string, targets []*ChaosTargetSelector, delay *time.Duration, percent int, duration *time.Duration, graceDuration *time.Duration) (*types.SuiteTest, error) {
	var steps []types.PlanStep

	s, err := composeIOLatencySteps(targets, delay, percent, duration)
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
			GracePeriod:  graceDuration,
		},
	}

	return test, nil
}

func ComposeNetworkLatencyTest(description string, targets []*ChaosTargetSelector, delay, jitter, duration, grace *time.Duration, correlation int) (*types.SuiteTest, error) {
	var steps []types.PlanStep
	s, err := composeNetworkLatencySteps(targets, delay, jitter, duration, correlation)
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
			GracePeriod:  grace,
		},
	}

	return test, nil
}
