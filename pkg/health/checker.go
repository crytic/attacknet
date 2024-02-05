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
	gracePeriod *time.Duration
}

func BuildHealthChecker(kubeClient *kubernetes.KubeClient, podsUnderTest []*chaos_mesh.PodUnderTest, healthCheckConfig confTypes.HealthCheckConfig) (*CheckOrchestrator, error) {
	networkType := "ethereum"

	var checkerImpl types.GenericNetworkChecker

	switch networkType {
	case "ethereum":
		a := ethereum.CreateEthNetworkChecker(kubeClient, podsUnderTest)
		checkerImpl = a
	default:
		log.Errorf("unknown network type: %s", networkType)
		return nil, stacktrace.NewError("unknown network type: %s", networkType)
	}
	return &CheckOrchestrator{checkerImpl: checkerImpl, gracePeriod: healthCheckConfig.GracePeriod}, nil
}

func (hc *CheckOrchestrator) RunChecks(ctx context.Context) (*types.HealthCheckResult, error) {
	start := time.Now()
	latestAllowable := start.Add(*hc.gracePeriod)
	log.Infof("Allowing up to %.0f seconds for health checks to pass on all nodes", hc.gracePeriod.Seconds())

	lastHealthCheckResult := &types.HealthCheckResult{}
	for {
		results, err := hc.checkerImpl.RunAllChecks(ctx, lastHealthCheckResult)
		if err != nil {
			return nil, err
		}
		lastHealthCheckResult = results
		if AllChecksPassed(results) {
			timeToPass := time.Since(start).Seconds()
			pctGraceUsed := timeToPass / hc.gracePeriod.Seconds() * 100
			log.Infof("Checks passed in %.0f seconds. Consumed %.1f pct of the %.0f second grace period", timeToPass, pctGraceUsed, hc.gracePeriod.Seconds())
			return results, nil
		}

		if time.Now().After(latestAllowable) {
			log.Warnf("Grace period elapsed and a health check is still failing. Time: %d", time.Now().Unix())
			return results, nil
		} else {
			log.Warn("Health checks failed but still in grace period")
			time.Sleep(1 * time.Second)
		}
	}
}

func AllChecksPassed(checks *types.HealthCheckResult) bool {
	if len(checks.LatestElBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(checks.LatestElBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(checks.FinalizedElBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(checks.FinalizedElBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(checks.LatestClBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(checks.LatestClBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}
	if len(checks.FinalizedClBlockResult.FailingClientsReportedBlock) > 0 {
		return false
	}
	if len(checks.FinalizedClBlockResult.FailingClientsReportedHash) > 0 {
		return false
	}

	return true
}
