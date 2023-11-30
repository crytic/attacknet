package pkg

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type EnclaveContextWrapper struct {
	Namespace              string
	kurtosisCtx            *kurtosis_context.KurtosisContext
	enclaveCtxInner        *enclaves.EnclaveContext
	reuseDevnetBetweenRuns bool
}

func (e *EnclaveContextWrapper) Destroy(ctx context.Context) {
	if e.reuseDevnetBetweenRuns {
		log.Infof("Skipping enclave deletion, enclave in namespace %s was flagged to be skip deletion", e.Namespace)
		return
	} else {
		log.Infof("Destroying enclave")
		err := e.kurtosisCtx.DestroyEnclave(ctx, e.enclaveCtxInner.GetEnclaveName())
		if err != nil {
			log.Fatal(err)
		}
	}
}

// pass-thru func. Figure out how to remove eventually.
func (e *EnclaveContextWrapper) RunStarlarkRemotePackageBlocking(
	ctx context.Context,
	packageId string,
	cfg *starlark_run_config.StarlarkRunConfig,
) (*enclaves.StarlarkRunResult, error) {
	return e.enclaveCtxInner.RunStarlarkRemotePackageBlocking(ctx, packageId, cfg)
}

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

func CreateEnclaveContext(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext, reuseDevnetBetweenRuns bool) (*EnclaveContextWrapper, error) {
	enclaveName := fmt.Sprintf("attacknet-%d", time.Now().Unix())
	enclaveCtx, err := kurtosisCtx.CreateEnclave(ctx, enclaveName)
	if err != nil {
		return nil, err
	}

	enclaveCtxWrapper := &EnclaveContextWrapper{
		Namespace:              fmt.Sprintf("kt-%s", enclaveCtx.GetEnclaveName()),
		kurtosisCtx:            kurtosisCtx,
		enclaveCtxInner:        enclaveCtx,
		reuseDevnetBetweenRuns: reuseDevnetBetweenRuns,
	}

	return enclaveCtxWrapper, nil
}

func CreateEnclaveFromExisting(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext, namespace string) (*EnclaveContextWrapper, error) {
	// strip the first 3 characters from the namespace ("kt-") to get the enclave name
	enclaveName := namespace[3:]

	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, enclaveName)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to locate kurtosis context by namespace %s", namespace)
	}

	enclaveCtxWrapper := &EnclaveContextWrapper{
		Namespace:              namespace,
		kurtosisCtx:            kurtosisCtx,
		enclaveCtxInner:        enclaveCtx,
		reuseDevnetBetweenRuns: true,
	}

	return enclaveCtxWrapper, nil
}

func StartNetwork(ctx context.Context, enclaveCtx *EnclaveContextWrapper, harnessConfig HarnessConfigParsed) error {
	log.Infof("------------ EXECUTING PACKAGE ---------------")
	cfg := &starlark_run_config.StarlarkRunConfig{
		SerializedParams: string(harnessConfig.NetworkConfig),
	}
	_, err := enclaveCtx.RunStarlarkRemotePackageBlocking(ctx, harnessConfig.NetworkPackage, cfg)
	if err != nil {
		return stacktrace.Propagate(err, "error occurred while running starklark package")
	} else {
		return nil
	}
}
