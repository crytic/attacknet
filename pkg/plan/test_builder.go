package plan

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"time"
)

func buildNodeClockSkewTest(description string, nodes []*network.Validator, skew, duration string) (*types.SuiteTest, error) {
	var steps []types.PlanStep
	for _, validator := range nodes {
		s, err := buildNodeClockSkewPlanSteps(validator, skew, duration)
		if err != nil {
			return nil, err
		}

		steps = append(steps, s...)
	}

	waitStep := buildWaitForFaultCompletionStep()
	steps = append(steps, *waitStep)

	test := &types.SuiteTest{
		TestName:  description,
		PlanSteps: steps,
		HealthConfig: types.HealthCheckConfig{
			EnableChecks: true,
			GracePeriod:  time.Second * 120,
		},
	}

	return test, nil
}
