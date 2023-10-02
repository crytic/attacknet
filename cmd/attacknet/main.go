package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"strings"
)

type Config struct {
	Blockchain struct {
		Name          string                 `json:"name"`
		Package       string                 `json:"package"`
		PackageConfig map[string]interface{} `json:"config"`
	} `json:"blockchain"`
}

func main() {
	// todo: use flag for arg parse

	//enclaveIdPrefix := "quick-start-go-example"
	//quickstartPackage := "github.com/kurtosis-tech/awesome-kurtosis/quickstart"
	defaultParallelism := int32(4)
	noDryRun := false
	//emptyPackageParams := "{}"
	//apiServiceName := "api"
	//contentType := "application/json"
	pathToMainFile := ""
	mainFunctionName := ""
	noExperimentalFeatureFlags := []kurtosis_core_rpc_api_bindings.KurtosisFeatureFlag{}

	ctx, cancelCtxFunc := context.WithCancel(context.Background())
	defer cancelCtxFunc()

	configFile := "config.json"
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	cfg := Config{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	//enclaveId := fmt.Sprintf("%s-%d", enclaveIdPrefix, time.Now().Unix())
	enclaveName := fmt.Sprintf("enclave-%s", cfg.Blockchain.Name)

	// todo: spawn kurtosis gateway?

	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		if strings.Contains(err.Error(), "connect: connection refused") {
			log.Fatal("Could not connect to the Kurtosis engine. Be sure the engine is running using `kurtosis engine status` or `kurtosis engine start`. You might also need to start the gateway using `kurtosis gateway`")
		} else {
			log.Fatal(err)
		}
	}

	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, enclaveName)
	if err == nil {
		// terminate enclave
		err := kurtosisCtx.DestroyEnclave(ctx, enclaveName)
		if err != nil {
			log.Fatal(err)
		}
	}

	enclaveCtx, err = kurtosisCtx.CreateEnclave(ctx, enclaveName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := kurtosisCtx.DestroyEnclave(ctx, enclaveName)
		if err != nil {
			log.Fatal(err)
		}
	}()

	pkgCfgJson, err := json.Marshal(cfg.Blockchain.PackageConfig)
	if err != nil {
		log.Fatal(err)
	}

	logrus.Info("------------ EXECUTING PACKAGE ---------------")
	starlarkRunResult, err := enclaveCtx.RunStarlarkRemotePackageBlocking(ctx, cfg.Blockchain.Package, pathToMainFile, mainFunctionName, string(pkgCfgJson), noDryRun, defaultParallelism, noExperimentalFeatureFlags)
	if err != nil {
		log.Fatal(err)
	}

	_ = starlarkRunResult
	_ = ctx
}
