package kurtosis

import (
	"attacknet/cmd/pkg/project"
	"context"
	"errors"
	"fmt"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
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

type PodUnderTest struct {
	Name        string
	Labels      map[string]string
	ExpectDeath bool
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
}

// pass-thru func. Figure out how to remove eventually.
func (e *EnclaveContextWrapper) RunStarlarkRemotePackageBlocking(
	ctx context.Context,
	packageId string,
	cfg *starlark_run_config.StarlarkRunConfig,
) (*enclaves.StarlarkRunResult, error) {
	return e.enclaveCtxInner.RunStarlarkRemotePackageBlocking(ctx, packageId, cfg)
}

// pass-thru func. Figure out how to remove eventually.
func (e *EnclaveContextWrapper) RunStarlarkRemotePackage(
	ctx context.Context,
	packageRootPath string,
	runConfig *starlark_run_config.StarlarkRunConfig,
) (chan *kurtosis_core_rpc_api_bindings.StarlarkRunResponseLine, context.CancelFunc, error) {
	return e.enclaveCtxInner.RunStarlarkRemotePackage(ctx, packageRootPath, runConfig)
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

func CreateOrImportContext(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext, cfg *project.ConfigParsed) (*EnclaveContextWrapper, bool, error) {
	enclaveName := getEnclaveName(cfg.AttacknetConfig.ExistingDevnetNamespace)

	// first check for existing enclave
	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, enclaveName)
	if err == nil {
		if !cfg.AttacknetConfig.ReuseDevnetBetweenRuns {
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

func StartNetwork(ctx context.Context, enclaveCtx *EnclaveContextWrapper, harnessConfig project.HarnessConfigParsed) error {
	log.Infof("------------ EXECUTING PACKAGE ---------------")
	cfg := &starlark_run_config.StarlarkRunConfig{
		SerializedParams: string(harnessConfig.NetworkConfig),
	}
	a, _, err := enclaveCtx.RunStarlarkRemotePackage(ctx, harnessConfig.NetworkPackage, cfg)
	if err != nil {
		return stacktrace.Propagate(err, "error running Starklark script")
	}

	// todo: clean this up when we decide to add log filtering
	progressIndex := 0
	for {
		t := <-a
		progress := t.GetProgressInfo()
		if progress != nil {
			progressMsgs := progress.CurrentStepInfo
			for i := progressIndex; i < len(progressMsgs); i++ {
				log.Infof("[Kurtosis] %s", progressMsgs[i])
			}
			progressIndex = len(progressMsgs)
		}

		info := t.GetInfo()
		if info != nil {
			log.Infof("[Kurtosis] %s", info.InfoMessage)
		}

		warn := t.GetWarning()
		if warn != nil {
			log.Warnf("[Kurtosis] %s", warn.WarningMessage)
		}

		e := t.GetError()
		if e != nil {
			log.Errorf("[Kurtosis] %s", e.String())
			return stacktrace.Propagate(errors.New("kurtosis deployment failed during execution"), "%s", e.String())
		}

		// ins := t.GetInstruction()
		// if ins != nil {
		// }

		insRes := t.GetInstructionResult()
		if insRes != nil {
			log.Infof("[Kurtosis] %s", insRes.SerializedInstructionResult)
		}

		finishRes := t.GetRunFinishedEvent()
		if finishRes != nil {
			log.Infof("[Kurtosis] %s", finishRes.GetSerializedOutput())
			if finishRes.IsRunSuccessful {
				log.Info("[Kurtosis] Devnet genesis successful. Passing back to Attacknet")
				return nil
			} else {
				log.Error("[Kurtosis] There was an error during genesis.")
				return stacktrace.Propagate(errors.New("kurtosis deployment failed"), "%s", finishRes.GetSerializedOutput())
			}
		}
	}
}
