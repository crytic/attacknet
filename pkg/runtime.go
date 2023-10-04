package pkg

import (
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

	cfg, err := LoadTestSuite(suiteFilePath)
	if err != nil {
		return err
	}

	// todo: spawn kurtosis gateway?

	kurtosisCtx, err := GetKurtosisContext()
	if err != nil {
		return err
	}

	enclaveCtx, err := CreateEnclaveContext(ctx, kurtosisCtx)
	namespace := fmt.Sprintf("kt-%s", enclaveCtx.GetEnclaveName())
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

	//namespace := "kt-attacknet-1696449504"
	stopCh, err := createGrafanaClient(ctx, namespace, cfg.AttacknetConfig)
	defer func() {
		close(stopCh)
	}()
	return err
}
