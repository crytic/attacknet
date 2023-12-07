package pkg

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"context"
	"errors"
	"time"

	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
)

func setupDevnet(ctx context.Context, cfg *ConfigParsed) (enclave *EnclaveContextWrapper, err error) {
	// todo: spawn kurtosis gateway?
	kurtosisCtx, err := GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	log.Infof("Creating a new Kurtosis enclave")
	enclaveCtx, _, err := CreateOrImportContext(ctx, kurtosisCtx, cfg)
	log.Infof("New enclave created under namespace %s", enclaveCtx.Namespace)
	if err != nil {
		return nil, err
	}

	log.Infof("Starting the blockchain genesis")
	err = StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
	if err != nil {
		return nil, err
	}
	return enclaveCtx, nil
}

func loadEnclaveFromExistingDevnet(ctx context.Context, cfg *ConfigParsed) (enclave *EnclaveContextWrapper, err error) {
	kurtosisCtx, err := GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	namespace := cfg.AttacknetConfig.ExistingDevnetNamespace
	log.Infof("Looking for existing enclave identified by namespace %s", namespace)
	enclaveCtx, enclaveCreated, err := CreateOrImportContext(ctx, kurtosisCtx, cfg)
	if err != nil {
		return nil, err
	}

	if enclaveCreated {
		// we need to genesis a new devnet regardless
		log.Info("Since we created a new kurtosis enclave, we must now genesis the blockchain.")
		err = StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
		if err != nil {
			return nil, err
		}
	} else {
		log.Infof("An active enclave matching %s was found", namespace)
	}

	return enclaveCtx, nil
}

func setupEnclave(ctx context.Context, cfg *ConfigParsed) (enclave *EnclaveContextWrapper, err error) {
	if cfg.AttacknetConfig.ExistingDevnetNamespace == "" {
		if cfg.AttacknetConfig.ReuseDevnetBetweenRuns {
			log.Warn("Could not re-use an existing devnet because no existingDevnetNamespace was set.")
		}
		enclave, err = setupDevnet(ctx, cfg)
	} else {
		enclave, err = loadEnclaveFromExistingDevnet(ctx, cfg)
	}
	return enclave, err
}

func StartTestSuite(ctx context.Context, cfg *ConfigParsed) error {
	enclave, err := setupEnclave(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		enclave.Destroy(ctx)
	}()

	// todo: move these into setupServices or something.
	log.Infof("Creating a Grafana client")
	grafanaTunnel, err := CreateGrafanaClient(ctx, enclave.Namespace, cfg.AttacknetConfig)
	if err != nil {
		return err
	}
	defer func() {
		grafanaTunnel.Cleanup()
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
		return err
	}

	// start core logic loop here.
	err = waitForInjectionCompleted(ctx, faultSession)
	if err != nil {
		return err
	}
	if faultSession.TestDuration != nil {
		durationSeconds := int(faultSession.TestDuration.Seconds())
		log.Infof("Fault injected successfully. Fault will run for %d seconds before recovering.", durationSeconds)
		time.Sleep(*faultSession.TestDuration)
	} else {
		log.Infof("Fault injected successfully. This fault has no specific duration.")
	}

	return waitForFaultRecovery(ctx, faultSession)
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
		errmsg := "chaos-mesh is still in a 'starting' state after 10 seconds. Something is probably wrong. Terminating"
		log.Error(errmsg)
		return stacktrace.NewError(errmsg)
	case chaos_mesh.Error:
		errmsg := "there was an unspecified error returned by chaos-mesh. inspect the fault resource"
		log.Error(errmsg)
		return stacktrace.NewError(errmsg)
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
