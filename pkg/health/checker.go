package health

import (
	chaosmesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health/ethereum"
	"attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

type CheckOrchestrator struct {
	checkerImpl types.HealthChecker
	gracePeriod *time.Duration
}

func BuildHealthChecker(
	networkType string,
	kubeClient *kubernetes.KubeClient,
	podsUnderTest []*chaosmesh.PodUnderTest,
	gracePeriod *time.Duration) (*CheckOrchestrator, error) {

	switch networkType {
	case "ethereum":
		return &CheckOrchestrator{
			checkerImpl: ethereum.CreateEthNetworkChecker(kubeClient, podsUnderTest),
			gracePeriod: gracePeriod,
		}, nil
	default:
		log.Errorf("unknown network type: %s", networkType)
		return nil, stacktrace.NewError("unknown network type: %s", networkType)
	}
}

func (co *CheckOrchestrator) RunChecksUntilPassOrGrace(ctx context.Context) (bool, interface{}, error) {
	start := time.Now()
	latestAllowable := start.Add(*co.gracePeriod)
	log.Infof("Allowing up to %.0f seconds for health checks to pass on all nodes", co.gracePeriod.Seconds())

	for {
		pass, err := co.checkerImpl.RunChecks(ctx)
		if err != nil {
			return false, nil, err
		}

		if pass {
			timeToPass := time.Since(start).Seconds()
			pctGraceUsed := timeToPass / co.gracePeriod.Seconds() * 100
			log.Infof("Checks passed in %.0f seconds. Consumed %.1f pct of the %.0f second grace period", timeToPass, pctGraceUsed, co.gracePeriod.Seconds())
			return true, co.checkerImpl.PopFinalResult(), nil
		}

		if time.Now().After(latestAllowable) {
			log.Warnf("Grace period elapsed and a health check is still failing. Time: %d", time.Now().Unix())
			return false, co.checkerImpl.PopFinalResult(), nil
		} else {
			log.Warn("Health checks failed but still in grace period")
			time.Sleep(1 * time.Second)
		}
	}
}
