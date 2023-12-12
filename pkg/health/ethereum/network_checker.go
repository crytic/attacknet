package ethereum

import (
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/kurtosis"
	log "github.com/sirupsen/logrus"
)
import "attacknet/cmd/pkg/health/types"

type EthNetworkChecker struct {
	namespace     string
	podsUnderTest []*kurtosis.PodUnderTest
}

func CreateEthNetworkChecker(namespace string, podsUnderTest []*kurtosis.PodUnderTest) *EthNetworkChecker {
	return &EthNetworkChecker{namespace: namespace, podsUnderTest: podsUnderTest}
}

func (e *EthNetworkChecker) RunAllChecks() ([]*types.CheckResult, error) {
	var alivePods []*kurtosis.PodUnderTest
	var alivePodNames []string

	// filter out pods expected dead
	for _, pod := range e.podsUnderTest {
		if !pod.ExpectDeath {
			alivePods = append(alivePods, pod)
			alivePodNames = append(alivePodNames, pod.Name)
		}
	}

	kubeConfig, _, err := kubernetes.CreateKubeClient()
	if err != nil {
		return nil, err
	}
	log.Infof("Starting port forward sessions to %d pods", len(alivePods))
	portForwardSessions, err := kubernetes.StartMultiPortForwards(alivePodNames, e.namespace, 3000, kubeConfig)
	if err != nil {
		return nil, err
	}
	log.Info("port forward sessions established")

	for _, session := range portForwardSessions {
		session.Close()
	}
	return nil, nil
}
