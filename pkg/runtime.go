package pkg

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	"attacknet/cmd/pkg/kurtosis"

	"attacknet/cmd/pkg/project"
	"context"
	"errors"
	"time"

	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
)

func StartTestSuite(ctx context.Context, cfg *project.ConfigParsed) error {
	enclave, err := setupEnclave(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		enclave.Destroy(ctx)
	}()

	a := make([]*kurtosis.PodUnderTest, 1)
	a[0] = &kurtosis.PodUnderTest{Name: "cl-3-prysm-geth", ExpectDeath: false, Labels: nil}
	log.Info("creating health checker")
	hc, err := health.BuildHealthChecker(cfg, enclave.Namespace, a)
	if err != nil {
		return err
	}
	_ = hc

	// todo: move these into setupServices or something.
	log.Infof("Creating a Grafana client")
	grafanaTunnel, err := CreateGrafanaClient(ctx, enclave.Namespace, cfg.AttacknetConfig)
	if err != nil {
		return err
	}
	defer func() {
		grafanaTunnel.Cleanup(false)
	}()

	// todo: set up grafana health checks/alerting here

	// todo: wait for finality or other network pre-conditions here.
	// probably check for initial health checks here too.

	//ds, err := grafanaTunnel.Client.GetDatasource(ctx, 1)
	//grafanaTunnel.Client.CreateAlertNotification()

	// create chaos-mesh client
	log.Infof("Creating a chaos-mesh client")
	chaosClient, err := chaos_mesh.CreateClient(enclave.Namespace)
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

	log.Infof("Starting fault injection")

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
	if faultSession.TestDuration != nil {
		durationSeconds := int(faultSession.TestDuration.Seconds())
		log.Infof("Fault injected successfully. Fault will run for %d seconds before recovering.", durationSeconds)
		time.Sleep(*faultSession.TestDuration)
	} else {
		log.Infof("Fault injected successfully. This fault has no specific duration.")
	}

	err = waitForFaultRecovery(ctx, faultSession)
	if err != nil {
		grafanaTunnel.Cleanup(true)
		return err
	}

	_, err = hc.RunChecksUntilTimeout()

	return err
}

func waitForInjectionCompleted(ctx context.Context, session *chaos_mesh.FaultSession) error {
	// First, wait 10 seconds to allow chaos-mesh to inject into the cluster.
	// If injection isn't complete after 10 seconds, something is  wrong and we should terminate.
	time.Sleep(10 * time.Second)

	status, err := session.GetStatus(ctx)
	if err != nil {
		return err
	}

	switch status {
	case chaos_mesh.InProgress:
		return nil
	case chaos_mesh.Stopping:
		errmsg := "fault changed to 'stopping' status after 10 seconds. faults must last longer than 10s"
		log.Error(errmsg)
		return stacktrace.NewError(errmsg)
	case chaos_mesh.Starting:
		var errmsg string
		if !session.TargetSelectionCompleted {
			errmsg = "chaos-mesh was unable to identify any pods for injection based on the configured criteria"
			log.Error(errmsg)
		} else {
			errmsg = "chaos-mesh is still in a 'starting' state after 10 seconds. Check kubernetes events to see what's wrong."
			log.Error(errmsg)
		}
		return stacktrace.NewError(errmsg)
	case chaos_mesh.Error:
		errmsg := "there was an unspecified error returned by chaos-mesh. inspect the fault resource"
		log.Error(errmsg)
		return stacktrace.NewError(errmsg)
	case chaos_mesh.Completed:
		// occurs for faults that perform an action immediately then terminate. (killing pods, etc)
		return nil
	default:
		return stacktrace.NewError("unknown chaos session state %s", status)
	}
}

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
