package pkg

import (
	"context"
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
	return nil
}
