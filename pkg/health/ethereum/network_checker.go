package ethereum

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	log "github.com/sirupsen/logrus"
)
import "attacknet/cmd/pkg/health/types"

type EthNetworkChecker struct {
	kubeClient          *kubernetes.KubeClient
	podsUnderTest       []*chaos_mesh.PodUnderTest
	podsUnderTestLookup map[string]*chaos_mesh.PodUnderTest
}

func CreateEthNetworkChecker(kubeClient *kubernetes.KubeClient, podsUnderTest []*chaos_mesh.PodUnderTest) *EthNetworkChecker {
	// convert podsUnderTest to a lookup
	podsUnderTestMap := make(map[string]*chaos_mesh.PodUnderTest)

	for _, pod := range podsUnderTest {
		podsUnderTestMap[pod.Name] = pod
	}

	return &EthNetworkChecker{
		podsUnderTest:       podsUnderTest,
		podsUnderTestLookup: podsUnderTestMap,
		kubeClient:          kubeClient,
	}
}

func (e *EthNetworkChecker) RunAllChecks(ctx context.Context) ([]*types.CheckResult, error) {
	labelKey := "kurtosistech.com.custom/ethereum-package.client-type"
	labelValue := "execution"

	var podsToHealthCheck []kubernetes.KubePod
	// add pods under test that match the label criteria
	for _, pod := range e.podsUnderTest {
		if pod.MatchesLabel(labelKey, labelValue) && !pod.ExpectDeath {
			podsToHealthCheck = append(podsToHealthCheck, pod)
		}
	}
	// add pods that were not targeted by a fault
	bystanders, err := e.kubeClient.PodsMatchingLabel(ctx, labelKey, labelValue)
	if err != nil {
		return nil, err
	}
	for _, pod := range bystanders {
		_, match := e.podsUnderTestLookup[pod.GetName()]
		// don't add pods we've already added
		if !match {
			podsToHealthCheck = append(podsToHealthCheck, pod)
		}
	}

	log.Infof("Starting port forward sessions to %d pods", len(podsToHealthCheck))
	portForwardSessions, err := e.kubeClient.StartMultiPortForwardToLabeledPods(
		podsToHealthCheck,
		labelKey,
		labelValue,
		3000)
	if err != nil {
		return nil, err
	}

	log.Info("Ready to query for health checks")

	for _, session := range portForwardSessions {
		session.Close()
	}
	return nil, nil
}
