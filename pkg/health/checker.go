package health

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health/ethereum"
	"attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/project"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
)

type CheckOrchestrator struct {
	checkerImpl types.GenericNetworkChecker
}

// todo: we may want to instantiate this at the beginning of the test suite to validat the configs, then update
// the podsUnderTest later.
func BuildHealthChecker(cfg *project.ConfigParsed, namespace string, podsUnderTest []*chaos_mesh.PodUnderTest) (*CheckOrchestrator, error) {
	networkType := cfg.HarnessConfig.NetworkType

	var checkerImpl types.GenericNetworkChecker

	switch networkType {
	case "ethereum":
		a := ethereum.CreateEthNetworkChecker(namespace, podsUnderTest)
		checkerImpl = a
	default:
		log.Errorf("unknown network type: %s", networkType)
		return nil, stacktrace.NewError("unknown network type: %s", networkType)
	}
	return &CheckOrchestrator{checkerImpl: checkerImpl}, nil
}

func (hc *CheckOrchestrator) RunChecksUntilTimeout() ([]*types.CheckResult, error) {
	return hc.checkerImpl.RunAllChecks()
}
