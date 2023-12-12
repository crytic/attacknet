package chaos_mesh

import "attacknet/cmd/pkg/kubernetes"

type PodUnderTest struct {
	Name           string
	Labels         map[string]string
	ExpectDeath    bool
	TouchedByFault bool
}

func (p *PodUnderTest) GetName() string {
	return p.Name
}

func (p *PodUnderTest) GetLabels() map[string]string {
	return p.Labels
}

func (p *PodUnderTest) MatchesLabel(key, value string) bool {
	v, exists := p.Labels[key]
	if !exists {
		return false
	} else {
		return v == value
	}
}

type PodUnderTestList []*PodUnderTest

func (pods PodUnderTestList) ToKubePods() []kubernetes.KubePod {
	kubePods := make([]kubernetes.KubePod, len(pods))
	for i, p := range pods {
		kubePods[i] = p
	}
	return kubePods
}

func GetPodsExpectedToBeDead(pods []*PodUnderTest) map[string]bool {
	expectation := make(map[string]bool)
	for _, pod := range pods {
		expectation[pod.Name] = pod.ExpectDeath
	}
	return expectation
}
