package kubernetes

type KubePod interface {
	GetName() string
	GetLabels() map[string]string
	MatchesLabel(string, string) bool
}

type KubePodList []KubePod

type Pod struct {
	Name   string
	Labels map[string]string
}

func (p *Pod) GetName() string {
	return p.Name
}

func (p *Pod) GetLabels() map[string]string {
	return p.Labels
}

func (p *Pod) MatchesLabel(key, value string) bool {
	v, exists := p.Labels[key]
	if !exists {
		return false
	} else {
		return v == value
	}
}
