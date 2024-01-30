package suite

import (
	"attacknet/cmd/pkg/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
	"strings"
	"time"
)

// We can't use chaos mesh's types because type-inlining is not supported in yaml.v3, making it so you can't serialize
// and deserialize to the same struct. Instead, we create pared down copies of the structs with no inlining.
// Completely scuffed.

type ChaosExpressionSelector struct {
	Key      string   `yaml:"key"`
	Operator string   `yaml:"operator"`
	Values   []string `yaml:"values"`
}

type Selector struct {
	LabelSelectors      map[string]string         `yaml:"labelSelectors,omitempty"`
	ExpressionSelectors []ChaosExpressionSelector `yaml:"expressionSelectors,omitempty"`
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
	Action   string `yaml:"action"`
}

type PodChaosFault struct {
	Spec       PodChaosSpec `yaml:"spec"`
	Kind       string       `yaml:"kind"`
	ApiVersion string       `yaml:"apiVersion"`
}

type PodChaosWrapper struct {
	PodChaosFault `yaml:"chaosFaultSpec"`
}

type IOChaosSpec struct {
	Selector `yaml:"selector"`
	Mode     string `yaml:"mode"`

	Action     string         `yaml:"action"`
	VolumePath string         `yaml:"volumePath"`
	Delay      *time.Duration `yaml:"delay"`
	Percent    int            `yaml:"percent"`
	Duration   *time.Duration `yaml:"duration"`
}

type IOChaosFault struct {
	Spec       IOChaosSpec `yaml:"spec"`
	Kind       string      `yaml:"kind"`
	ApiVersion string      `yaml:"apiVersion"`
}

type IOChaosWrapper struct {
	IOChaosFault `yaml:"chaosFaultSpec"`
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

func buildClockSkewFault(description, timeOffset, duration string, expressionSelectors []ChaosExpressionSelector) (*types.PlanStep, error) {

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

func buildPodRestartFault(description string, expressionSelectors []ChaosExpressionSelector) (*types.PlanStep, error) {
	t := PodChaosWrapper{
		PodChaosFault: PodChaosFault{
			Kind:       "PodChaos",
			ApiVersion: "chaos-mesh.org/v1alpha1",
			Spec: PodChaosSpec{
				Duration: "5s",
				Mode:     "all",
				Action:   "pod-failure",
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

func getVolumePathForIOFault(podName string) (string, error) {
	var nodeType string
	parts := strings.Split(podName, "-")
	if parts[0] == "el" {
		nodeType = "execution"
	} else {
		nodeType = "consensus"
	}
	if parts[len(parts)-1] == "validator" {
		return "", stacktrace.NewError("cannot create an i/o latency fault on a validator sidecar pod. Try to target matching clients only: %s", podName)
	}
	clientName := parts[2]
	volumeTarget := fmt.Sprintf("/data/%s/%s-data", clientName, nodeType)
	return volumeTarget, nil
}

func buildIOLatencyFault(description string, expressionSelector ChaosExpressionSelector, delay *time.Duration, percent int, duration *time.Duration) ([]types.PlanStep, error) {
	var steps []types.PlanStep

	for _, podName := range expressionSelector.Values {
		volumePath, err := getVolumePathForIOFault(podName)
		if err != nil {
			return nil, err
		}

		t := IOChaosWrapper{
			IOChaosFault: IOChaosFault{
				Kind:       "IOChaos",
				ApiVersion: "chaos-mesh.org/v1alpha1",
				Spec: IOChaosSpec{
					Duration: duration,
					Mode:     "all",
					Selector: Selector{
						ExpressionSelectors: []ChaosExpressionSelector{expressionSelector},
					},
					Action:     "latency",
					VolumePath: volumePath,
					Delay:      delay,
					Percent:    percent,
				},
			},
		}

		faultSpec, err := convertFaultSpecToMap(t)
		if err != nil {
			return nil, err
		}

		step := types.PlanStep{
			StepType:        types.InjectFault,
			StepDescription: description,
			Spec:            faultSpec,
		}
		steps = append(steps, step)
	}

	return steps, nil
}
