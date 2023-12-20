package health

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health/ethereum"
	"attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	confTypes "attacknet/cmd/pkg/types"
	"context"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

type CheckOrchestrator struct {
	checkerImpl types.GenericNetworkChecker
	gracePeriod time.Duration
}

func BuildHealthChecker(cfg *confTypes.ConfigParsed, kubeClient *kubernetes.KubeClient, podsUnderTest []*chaos_mesh.PodUnderTest, healthCheckConfig confTypes.HealthCheckConfig) (*CheckOrchestrator, error) {
	networkType := cfg.HarnessConfig.NetworkType

	var checkerImpl types.GenericNetworkChecker

	switch networkType {
	case "ethereum":
		a := ethereum.CreateEthNetworkChecker(kubeClient, podsUnderTest)
		checkerImpl = a
	default:
		log.Errorf("unknown network type: %s", networkType)
		return nil, stacktrace.NewError("unknown network type: %s", networkType)
	}
	return &CheckOrchestrator{checkerImpl: checkerImpl}, nil
}

func (hc *CheckOrchestrator) RunChecks(ctx context.Context) ([]*types.CheckResult, error) {
	latestAllowable := time.Now().Add(hc.gracePeriod)
	log.Infof("Allowing up to %.0f seconds for health checks to pass on all nodes", hc.gracePeriod.Seconds())

	for {
		results, err := hc.checkerImpl.RunAllChecks(ctx)
		if err != nil {
			return nil, err
		}
		if AllChecksPassed(results) {
			return results, nil
		}

		if time.Now().After(latestAllowable) {
			log.Warn("Grace period elapsed and a health check is still failing")
			return results, nil
		} else {
			log.Warn("Health checks failed but still in grace period")
			time.Sleep(1 * time.Second)
		}
	}
}

func AllChecksPassed(checks []*types.CheckResult) bool {
	for _, r := range checks {
		if len(r.PodsFailing) != 0 {
			return false
		}
	}
	return true
}