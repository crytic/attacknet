package pkg

import (
	chaosmesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/runtime"
	"attacknet/cmd/pkg/test_executor"
	"attacknet/cmd/pkg/types"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

func StartTestSuite(ctx context.Context, cfg *types.ConfigParsed) error {
	enclave, err := runtime.SetupEnclave(ctx, cfg)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.CreateKubeClient(enclave.Namespace)
	if err != nil {
		return err
	}

	// create chaos-mesh client
	log.Infof("Creating a chaos-mesh client")
	chaosClient, err := chaosmesh.CreateClient(enclave.Namespace, kubeClient)
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

	healthArtifactSerializer, err := health.BuildArtifactSerializer(cfg.HarnessConfig.NetworkType)
	if err != nil {
		return err
	}

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

			hc, err := health.BuildHealthChecker(
				cfg.HarnessConfig.NetworkType,
				kubeClient,
				podsUnderTest,
				test.HealthConfig.GracePeriod)
			if err != nil {
				return err
			}
			passing, resultDetail, err := hc.RunChecksUntilPassOrGrace(ctx)
			if err != nil {
				return err
			}

			err = healthArtifactSerializer.AddHealthCheckResult(resultDetail, podsUnderTest, test)
			if err != nil {
				return err
			}

			if !passing {
				log.Warn("Some health checks failed. Stopping test suite.")
				break
			}
		} else {
			log.Info("Skipping health checks")
		}
	}

	healthArtifact, err := healthArtifactSerializer.SerializeArtifacts()
	if err != nil {
		return err
	}

	// write
	artifactFilename := fmt.Sprintf("results-%d.yaml", time.Now().UnixMilli())
	err = WriteFileOnSubpath("artifacts", artifactFilename, healthArtifact)
	if err != nil {
		return err
	}

	enclave.Destroy(ctx)
	return nil
}
