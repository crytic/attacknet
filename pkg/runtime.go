package pkg

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/test_executor"
	"attacknet/cmd/pkg/types"
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func StartTestSuite(ctx context.Context, cfg *types.ConfigParsed) error {
	enclave, err := setupEnclave(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		enclave.Destroy(ctx)
	}()

	kubeClient, err := kubernetes.CreateKubeClient(enclave.Namespace)
	if err != nil {
		return err
	}

	// todo: move this into setupServices or something.
	log.Infof("Creating a Grafana client")
	grafanaTunnel, err := CreateGrafanaClient(ctx, kubeClient, cfg.AttacknetConfig)
	if err != nil {
		return err
	}
	defer func() {
		grafanaTunnel.Cleanup(false)
	}()

	// create chaos-mesh client
	log.Infof("Creating a chaos-mesh client")
	chaosClient, err := chaos_mesh.CreateClient(enclave.Namespace, kubeClient)
	if err != nil {
		grafanaTunnel.Cleanup(true)
		return err
	}

	// standby for timer
	log.Infof(
		"Waiting %d seconds before starting fault injection",
		cfg.AttacknetConfig.WaitBeforeInjectionSeconds,
	)
	time.Sleep(time.Duration(cfg.AttacknetConfig.WaitBeforeInjectionSeconds) * time.Second)

	log.Infof("Running %d tests", len(cfg.TestConfig.Tests))

	for i, test := range cfg.TestConfig.Tests {
		log.Infof("Running test #%d, '%s'", i, test.TestName)
		executor := test_executor.CreateTestExecutor(chaosClient, test)

		err = executor.RunTestPlan(ctx)
		if err != nil {
			log.Errorf("Error while running test #%d", i+1)
			return err
		} else {
			log.Infof("Test #%d completed.", i+1)
		}

		if test.HealthConfig.EnableChecks {
			podsUnderTest, err := executor.GetPodsUnderTest()
			if err != nil {
				return err
			}

			hc, err := health.BuildHealthChecker(cfg, kubeClient, podsUnderTest, test.HealthConfig)
			if err != nil {
				return err
			}
			results, err := hc.RunChecks(ctx)
			if err != nil {
				return err
			}
			// todo: log here
			_ = results
		}
	}

	return nil
	/*
		faultSession, err := chaosClient.StartFault(ctx, cfg.Tests[0].FaultSpec)
		if err != nil {
			grafanaTunnel.Cleanup(true)
			return err
		}

		// start core logic loop here.
		err = waitForInjectionCompleted(ctx, faultSession)
		if err != nil {
			grafanaTunnel.Cleanup(true)
			return err
		}
		var timeToSleep time.Duration
		if faultSession.TestDuration != nil {
			durationSeconds := int(faultSession.TestDuration.Seconds())
			log.Infof("Fault injected successfully. Fault will run for %d seconds before recovering.", durationSeconds)
			timeToSleep = *faultSession.TestDuration
		} else {
			log.Infof("Fault injected successfully. This fault has no specific duration.")
		}
		time.Sleep(timeToSleep)

		// we can build the health checker once the fault is injected
		log.Info("creating health checker")
		hc, err := health.BuildHealthChecker(cfg, kubeClient, faultSession.PodsUnderTest)
		if err != nil {
			return err
		}
		_ = hc

		err = waitForFaultRecovery(ctx, faultSession)
		if err != nil {
			grafanaTunnel.Cleanup(true)
			return err
		}

		_, err = hc.RunChecksUntilTimeout(ctx)

		return err*/
}

// todo: move to fault session?
/*


func waitForFaultRecovery(ctx context.Context, session *chaos_mesh.FaultSession) error {
	for {
		status, err := session.GetStatus(ctx)
		if err != nil {
			return err
		}

		switch status {
		case chaos_mesh.InProgress:
			log.Infof("The fault is still finishing up. Sleeping for 10s")
			time.Sleep(10 * time.Second)
		case chaos_mesh.Stopping:
			log.Infof("The fault is being stopped. Sleeping for 10s")
			time.Sleep(10 * time.Second)
		case chaos_mesh.Error:
			log.Errorf("there was an error returned by chaos-mesh")
			return errors.New("there was an unspecified error returned by chaos-mesh. inspect the fault resource")
		case chaos_mesh.Completed:
			log.Infof("The fault terminated successfully!")
			return nil
		default:
			return stacktrace.NewError("unknown chaos session state %s", status)
		}
		// todo: add timeout break if no changes in k8s resource after fault duration elapses
	}
}
*/
