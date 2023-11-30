package pkg

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"os"
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
	} else {
		log.Infof("Destroying enclave")
		err := e.kurtosisCtx.DestroyEnclave(ctx, e.enclaveCtxInner.GetEnclaveName())
		if err != nil {
			log.Fatal(err)
		}
	}
	return
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

func getEnclaveName(namespace string) string {
	var enclaveName string
	if namespace != "" {
		enclaveName = namespace[3:]
	} else {
		enclaveName = fmt.Sprintf("attacknet-%d", time.Now().Unix())
	}
	return enclaveName
}

func isErrorNoEnclaveFound(err error) bool {
	rootCause := stacktrace.RootCause(err)
	if strings.Contains(rootCause.Error(), "Couldn't find an enclave for identifier") {
		return true
	} else {
		return false
	}
}

func CreateOrImportContext(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext, cfg *ConfigParsed) (*EnclaveContextWrapper, bool, error) {
	enclaveName := getEnclaveName(cfg.AttacknetConfig.ExistingDevnetNamespace)

	// first check for existing enclave
	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, enclaveName)
	if err == nil {
		if cfg.AttacknetConfig.ReuseDevnetBetweenRuns == false {
			log.Errorf("An existing enclave was found with the name %s, but ReuseDevnetBetweenRuns is set to false. Todo: add tear-down logic here.", enclaveName)
			os.Exit(1)
		}
		enclaveCtxWrapper := &EnclaveContextWrapper{
			Namespace:              fmt.Sprintf("kt-%s", enclaveCtx.GetEnclaveName()),
			kurtosisCtx:            kurtosisCtx,
			enclaveCtxInner:        enclaveCtx,
			reuseDevnetBetweenRuns: true,
		}
		return enclaveCtxWrapper, false, nil
	} else {
		// check if no enclave found
		if !isErrorNoEnclaveFound(err) {
			return nil, false, err
		}

		log.Infof("No existing kurtosis enclave by the name of %s was found. Creating a new one.", enclaveName)
		enclaveCtxNew, err := kurtosisCtx.CreateEnclave(ctx, enclaveName)
		if err != nil {
			return nil, false, err
		}
		enclaveCtxWrapper := &EnclaveContextWrapper{
			Namespace:              fmt.Sprintf("kt-%s", enclaveCtxNew.GetEnclaveName()),
			kurtosisCtx:            kurtosisCtx,
			enclaveCtxInner:        enclaveCtxNew,
			reuseDevnetBetweenRuns: cfg.AttacknetConfig.ReuseDevnetBetweenRuns,
		}
		return enclaveCtxWrapper, true, nil
	}
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
