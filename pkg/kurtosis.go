package pkg

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
	"time"
)

func GetKurtosisContext() (*kurtosis_context.KurtosisContext, error) {
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		if strings.Contains(err.Error(), "connect: connection refused") {
			return nil, fmt.Errorf("could not connect to the Kurtosis engine. Be sure the engine is running using `kurtosis engine status` or `kurtosis engine start`. You might also need to start the gateway using `kurtosis gateway` - %w", err)
		} else {
			return nil, err
		}
	}
	return kurtosisCtx, nil
}

func CreateEnclaveContext(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext) (*enclaves.EnclaveContext, error) {
	enclaveName := fmt.Sprintf("attacknet-%d", time.Now().Unix())
	enclaveCtx, err := kurtosisCtx.CreateEnclave(ctx, enclaveName)
	if err != nil {
		return nil, err
	}

	return enclaveCtx, nil
}

func StartNetwork(ctx context.Context, enclaveCtx *enclaves.EnclaveContext, harnessConfig HarnessConfigParsed) {
	logrus.Info("------------ EXECUTING PACKAGE ---------------")
	_, err := enclaveCtx.RunStarlarkRemotePackageBlocking(ctx, harnessConfig.NetworkPackage, "", "", string(harnessConfig.NetworkConfig), false, 4, []kurtosis_core_rpc_api_bindings.KurtosisFeatureFlag{})
	if err != nil {
		log.Fatal(err)
	}
}
