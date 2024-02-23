package runtime

import (
	"attacknet/cmd/pkg/kurtosis"
	"attacknet/cmd/pkg/types"
	"context"
	"github.com/sirupsen/logrus"
)

func setupDevnet(ctx context.Context, cfg *types.ConfigParsed) (enclave *kurtosis.EnclaveContextWrapper, err error) {
	// todo: spawn kurtosis gateway?
	kurtosisCtx, err := kurtosis.GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Creating a new Kurtosis enclave")
	enclaveCtx, _, err := kurtosis.CreateOrImportContext(ctx, kurtosisCtx, cfg)
	logrus.Infof("New enclave created under namespace %s", enclaveCtx.Namespace)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Starting the blockchain genesis")
	err = kurtosis.StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
	if err != nil {
		return nil, err
	}

	return enclaveCtx, nil
}

func loadEnclaveFromExistingDevnet(ctx context.Context, cfg *types.ConfigParsed) (enclave *kurtosis.EnclaveContextWrapper, err error) {
	kurtosisCtx, err := kurtosis.GetKurtosisContext()
	if err != nil {
		return nil, err
	}

	namespace := cfg.AttacknetConfig.ExistingDevnetNamespace
	logrus.Infof("Looking for existing enclave identified by namespace %s", namespace)
	enclaveCtx, enclaveCreated, err := kurtosis.CreateOrImportContext(ctx, kurtosisCtx, cfg)
	if err != nil {
		return nil, err
	}

	if enclaveCreated {
		// we need to genesis a new devnet regardless
		logrus.Info("Since we created a new kurtosis enclave, we must now genesis the blockchain.")
		err = kurtosis.StartNetwork(ctx, enclaveCtx, cfg.HarnessConfig)
		if err != nil {
			return nil, err
		}
	} else {
		logrus.Infof("An active enclave matching %s was found", namespace)
	}

	return enclaveCtx, nil
}

func SetupEnclave(ctx context.Context, cfg *types.ConfigParsed) (enclave *kurtosis.EnclaveContextWrapper, err error) {
	if cfg.AttacknetConfig.ExistingDevnetNamespace == "" {
		if cfg.AttacknetConfig.ReuseDevnetBetweenRuns {
			logrus.Warn("Could not re-use an existing devnet because no existingDevnetNamespace was set.")
		}
		enclave, err = setupDevnet(ctx, cfg)
	} else {
		enclave, err = loadEnclaveFromExistingDevnet(ctx, cfg)
	}
	return enclave, err
}
