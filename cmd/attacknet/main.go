package main

import (
	"attacknet/cmd/pkg"
	"context"
	"log"
)

func main() {
	// todo: use flag for arg parse

	ctx, cancelCtxFunc := context.WithCancel(context.Background())
	defer cancelCtxFunc()

	configFile := "/Users/bsamuels/projects/attacknet/config.yaml"

	cfg, err := pkg.LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// todo: spawn kurtosis gateway?

	kurtosisCtx, err := pkg.GetKurtosisContext()
	if err != nil {
		log.Fatal(err)
	}

	enclaveCtx, err := pkg.CreateEnclaveContext(ctx, kurtosisCtx)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := kurtosisCtx.DestroyEnclave(ctx, enclaveCtx.GetEnclaveName())
		if err != nil {
			log.Fatal(err)
		}
	}()

	pkg.StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
}
