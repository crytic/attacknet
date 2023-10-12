package pkg

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func StartTestSuite(ctx context.Context, suiteName string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	suiteName = suiteName + ".yaml"
	suiteFilePath := filepath.Join(dir, suiteDirectory, suiteName)

	dummyNetwork := true

	cfg, err := LoadTestSuite(suiteFilePath)
	if err != nil {
		return err
	}

	// todo: spawn kurtosis gateway?
	var namespace string
	if !dummyNetwork {
		kurtosisCtx, err := GetKurtosisContext()
		if err != nil {
			return err
		}

		enclaveCtx, err := CreateEnclaveContext(ctx, kurtosisCtx)
		namespace = fmt.Sprintf("kt-%s", enclaveCtx.GetEnclaveName())
		if err != nil {
			return err
		}

		defer func() {
			err := kurtosisCtx.DestroyEnclave(ctx, enclaveCtx.GetEnclaveName())
			if err != nil {
				log.Fatal(err)
			}
		}()

		StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
	} else {
		namespace = "kt-ethereum"
	}

	grafanaTunnel, err := CreateGrafanaClient(ctx, namespace, cfg.AttacknetConfig)
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

	//_ = ds

	// create chaos-mesh client
	chaosClient, err := chaos_mesh.CreateClient(namespace)
	if err != nil {
		return err
	}

	faultSession, err := chaosClient.StartFault(ctx, cfg.Tests[0].FaultSpec)
	//time.Sleep(5 * time.Second)
	_, _ = faultSession.GetStatus(ctx)

	_, _ = faultSession.GetStatus(ctx)

	_, _ = faultSession.GetStatus(ctx)

	return err
}
