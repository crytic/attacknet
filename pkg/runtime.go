package pkg

import (
	"attacknet/cmd/pkg/artifacts"
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/test_executor"
	"attacknet/cmd/pkg/types"
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func StartTestSuite(ctx context.Context, cfg *types.ConfigParsed) error {
	enclave, err := setupEnclave(ctx, cfg)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.CreateKubeClient(enclave.Namespace)
	if err != nil {
		return err
	}

	// create chaos-mesh client
	log.Infof("Creating a chaos-mesh client")
	chaosClient, err := chaos_mesh.CreateClient(enclave.Namespace, kubeClient)
	if err != nil {
		//	grafanaTunnel.Cleanup(true)
		return err
	}

	// standby for timer
	log.Infof(
		"Waiting %d seconds before starting fault injection",
		cfg.AttacknetConfig.WaitBeforeInjectionSeconds,
	)
	time.Sleep(time.Duration(cfg.AttacknetConfig.WaitBeforeInjectionSeconds) * time.Second)

	log.Infof("Running %d tests", len(cfg.TestConfig.Tests))

	var testArtifacts []*artifacts.TestArtifact

	for i, test := range cfg.TestConfig.Tests {
		log.Infof("Running test (%d/%d): '%s'", i+1, len(cfg.TestConfig.Tests), test.TestName)
		executor := test_executor.CreateTestExecutor(chaosClient, test)

		err = executor.RunTestPlan(ctx)
		if err != nil {
			log.Errorf("Error while running test #%d", i+1)
			return err
		} else {
			log.Infof("Test #%d steps completed.", i+1)
		}

		if test.HealthConfig.EnableChecks {
			log.Info("Starting health checks")
			podsUnderTest, err := executor.GetPodsUnderTest()
			if err != nil {
				return err
			}

			hc, err := health.BuildHealthChecker(cfg, kubeClient, podsUnderTest, test.HealthConfig)
			if err != nil {
				return err
			}
			results, err := hc.RunChecks(ctx)
			if err != nil {
				return err
			}
			testArtifact := artifacts.BuildTestArtifact(results, podsUnderTest, test)
			testArtifacts = append(testArtifacts, testArtifact)
			if !testArtifact.TestPassed {
				log.Warn("Some health checks failed. Stopping test suite.")
				break
			}
		} else {
			log.Info("Skipping health checks")
		}
	}
	err = artifacts.SerializeTestArtifacts(testArtifacts)
	if err != nil {
		return err
	}

	enclave.Destroy(ctx)

	return nil
}
