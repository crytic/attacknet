package pkg

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"context"
	"errors"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

func setupDevnet(ctx context.Context, harnessCfg HarnessConfigParsed) (enclave *EnclaveContextWrapper, err error) {
	// todo: spawn kurtosis gateway?
	kurtosisCtx, err := GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	log.Infof("Creating a new Kurtosis enclave")
	enclaveCtx, err := CreateEnclaveContext(ctx, kurtosisCtx)
	log.Infof("New enclave created under namespace %s", enclaveCtx.Namespace)
	if err != nil {
		return nil, err
	}

	log.Infof("Starting the blockchain genesis")
	err = StartNetwork(ctx, enclaveCtx, harnessCfg)
	if err != nil {
		return nil, err
	}
	return enclaveCtx, nil
}

func loadEnclaveFromExistingDevnet(ctx context.Context, attacknetCfg AttacknetConfig) (enclave *EnclaveContextWrapper, err error) {
	kurtosisCtx, err := GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	log.Infof("Looking for existing enclave identified by namespace %s", attacknetCfg.ExistingDevnetNamespace)
	enclaveCtx, err := CreateEnclaveFromExisting(ctx, kurtosisCtx, attacknetCfg.ExistingDevnetNamespace)
	if err != nil {
		return nil, err
	} else {
		return enclaveCtx, nil
	}
}

func setupEnclave(ctx context.Context, cfg *ConfigParsed) (enclave *EnclaveContextWrapper, err error) {
	if cfg.AttacknetConfig.ExistingDevnetNamespace == "" {
		enclave, err = setupDevnet(ctx, cfg.HarnessConfig)
	} else {
		enclave, err = loadEnclaveFromExistingDevnet(ctx, cfg.AttacknetConfig)
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
		close(grafanaTunnel.PortForwardStopCh)
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
	status, err := faultSession.GetStatus(ctx)
	if err != nil {
		return err
	}
	if status == chaos_mesh.Starting || status == chaos_mesh.InProgress {
		duration, err := faultSession.GetDuration(ctx)
		if err != nil {
			return err
		}
		log.Infof("Fault injected successfully. Fault will run for %s before recovering.", duration)
	} else {
		return stacktrace.Propagate(errors.New("something went wrong during fault injection that didn't raise any Go errors"), "status: %s", status)
	}

	// start core logic loop here.
	for {
		time.Sleep(10 * time.Second)
		status, err := faultSession.GetStatus(ctx)
		if err != nil {
			return err
		}

		if status == chaos_mesh.InProgress {
			log.Infof("The fault is still running. Sleeping for 10s")

		}

		if status == chaos_mesh.Stopping {
			log.Infof("The fault is being stopped")
		}

		if status == chaos_mesh.Completed {
			log.Infof("The fault terminated successfully!")
			break
		}

		if status == chaos_mesh.Starting {
			msg := "chaos-mesh is still in a 'starting' state after 10 seconds. Something is probably wrong. Terminating"
			log.Errorf(msg)
			return errors.New(msg)
		}
		if status == chaos_mesh.Error {
			log.Errorf("there was an error returned by chaos-mesh")
			return errors.New("there was an unspecified error returned by chaos-mesh. inspect the fault resource")
		}
		// todo: add timeout break if no changes in k8s resource after fault duration elapses
	}

	return nil
}
