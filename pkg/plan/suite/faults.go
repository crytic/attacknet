package suite

import (
	"attacknet/cmd/pkg/types"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
)

// We can't use chaos mesh's types because type-inlining is not supported in yaml.v3, making it so you can't serialize
// and deserialize to the same struct. Instead, we create pared down copies of the structs with no inlining.
// Completely scuffed.

type ExpressionSelector struct {
	Key      string   `yaml:"key"`
	Operator string   `yaml:"operator"`
	Values   []string `yaml:"values"`
}

type Selector struct {
	LabelSelectors      map[string]string    `yaml:"labelSelectors,omitempty"`
	ExpressionSelectors []ExpressionSelector `yaml:"expressionSelectors,omitempty"`
}

type TimeChaosSpec struct {
	Selector   `yaml:"selector"`
	Mode       string `yaml:"mode"`
	Action     string `yaml:"action"`
	TimeOffset string `yaml:"timeOffset"`
	Duration   string `yaml:"duration"`
}

type TimeChaosFault struct {
	Spec       TimeChaosSpec `yaml:"spec"`
	Kind       string        `yaml:"kind"`
	ApiVersion string        `yaml:"apiVersion"`
}

type TimeChaosWrapper struct {
	TimeChaosFault `yaml:"chaosFaultSpec"`
}

type PodChaosSpec struct {
	Selector `yaml:"selector"`
	Mode     string `yaml:"mode"`
	Duration string `yaml:"duration"`
}

type PodChaosFault struct {
	Spec       PodChaosSpec `yaml:"spec"`
	Kind       string       `yaml:"kind"`
	ApiVersion string       `yaml:"apiVersion"`
}

type PodChaosWrapper struct {
	PodChaosFault `yaml:"chaosFaultSpec"`
}

func convertFaultSpecToMap(s interface{}) (map[string]interface{}, error) {
	// convert to map[string]interface{} using yaml intermediate. seriously.
	bs, err := yaml.Marshal(s)
	if err != nil {
		return nil, stacktrace.Propagate(err, "intermediate yaml marshalling failed")
	}

	var faultSpec map[string]interface{}
	err = yaml.Unmarshal(bs, &faultSpec)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to deserialize intermediate yaml")
	}
	return faultSpec, nil
}

func buildClockSkewFault(description, timeOffset, duration string, expressionSelectors []ExpressionSelector) (*types.PlanStep, error) {

	t := TimeChaosWrapper{
		TimeChaosFault: TimeChaosFault{
			Kind:       "TimeChaos",
			ApiVersion: "chaos-mesh.org/v1alpha1",
			Spec: TimeChaosSpec{
				Duration:   duration,
				TimeOffset: timeOffset,
				Mode:       "all",
				Action:     "delay",
				Selector: Selector{
					ExpressionSelectors: expressionSelectors,
				},
			},
		},
	}

	faultSpec, err := convertFaultSpecToMap(t)
	if err != nil {
		return nil, err
	}

	step := &types.PlanStep{
		StepType:        types.InjectFault,
		StepDescription: description,
		Spec:            faultSpec,
	}
	return step, nil
}

func buildPodRestartFault(description string, expressionSelectors []ExpressionSelector) (*types.PlanStep, error) {
	t := PodChaosWrapper{
		PodChaosFault: PodChaosFault{
			Kind:       "PodChaos",
			ApiVersion: "chaos-mesh.org/v1alpha1",
			Spec: PodChaosSpec{
				Duration: "10s",
				Mode:     "all",
				Selector: Selector{
					ExpressionSelectors: expressionSelectors,
				},
			},
		},
	}

	faultSpec, err := convertFaultSpecToMap(t)
	if err != nil {
		return nil, err
	}

	step := &types.PlanStep{
		StepType:        types.InjectFault,
		StepDescription: description,
		Spec:            faultSpec,
	}
	return step, nil
}
