package exploration

import (
	"attacknet/cmd/pkg/artifacts"
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/plan"
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/plan/suite"
	"attacknet/cmd/pkg/test_executor"
	"attacknet/cmd/pkg/types"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const WaitBetweenTestsSecs = 60
const Seed = 555

func getRandomAttackSize() suite.AttackSize {
	//return suite.AttackOne
	sizes := []suite.AttackSize{
		suite.AttackOne,
		suite.AttackAll,
		suite.AttackMajority,
		suite.AttackMinority,
		suite.AttackSupermajority,
		suite.AttackSuperminority,
	}
	i := len(sizes)
	return sizes[rand.Intn(i)]
}

func getTargetSpec() suite.TargetingSpec {
	targetSpecs := []suite.TargetingSpec{
		suite.TargetMatchingClient,
		suite.TargetMatchingNode,
	}
	_ = targetSpecs
	//return suite.TargetMatchingClient
	return targetSpecs[rand.Intn(2)]
}

func buildRandomLatencyTest(targetDescription string, targetSelectors []*suite.ChaosTargetSelector) (*types.SuiteTest, error) {
	minDelayMilliSeconds := 10
	maxDelayMilliSeconds := 1000
	minDurationSeconds := 10
	maxDurationSeconds := 1000
	minJitterMilliseconds := 10
	maxJitterMilliseconds := 1000
	minCorrelation := 0
	maxCorrelation := 100

	grace := time.Second * 300
	duration := time.Second * time.Duration(rand.Intn(maxDurationSeconds-minDurationSeconds)+minDurationSeconds)
	delay := time.Millisecond * time.Duration(rand.Intn(maxDelayMilliSeconds-minDelayMilliSeconds)+minDelayMilliSeconds)
	jitter := time.Millisecond * time.Duration(rand.Intn(maxJitterMilliseconds-minJitterMilliseconds)+minJitterMilliseconds)
	correlation := rand.Intn(maxCorrelation-minCorrelation) + minCorrelation
	description := fmt.Sprintf("Apply %s network latency for %s. Jitter: %s, correlation: %d against %d targets. %s", delay, duration, jitter, correlation, len(targetSelectors), targetDescription)
	log.Info(description)
	return suite.ComposeNetworkLatencyTest(
		description,
		targetSelectors,
		&delay,
		&jitter,
		&duration,
		&grace,
		correlation,
	)
}

func pickRandomClient(config *plan.PlannerConfig) (string, bool) {
	//return "reth", true
	isExec := rand.Intn(2)
	if isExec == 1 {
		numExecClients := len(config.ExecutionClients)
		idx := rand.Intn(numExecClients)
		return config.ExecutionClients[idx].Name, true
	} else {
		numBeaconClients := len(config.ConsensusClients)
		idx := rand.Intn(numBeaconClients)
		return config.ConsensusClients[idx].Name, false
	}
}

func StartExploration(config *plan.PlannerConfig) error {
	// todo: big refactor
	ctx, cancelCtxFunc := context.WithCancel(context.Background())
	defer cancelCtxFunc()
	nodes, err := network.ComposeNetworkTopology(
		config.Topology,
		config.FaultConfig.TargetClient,
		config.ExecutionClients,
		config.ConsensusClients,
	)
	if err != nil {
		return err
	}
	testableNodes := nodes[1:]

	// dedupe from runtime?
	kubeClient, err := kubernetes.CreateKubeClient(config.KubernetesNamespace)
	if err != nil {
		return err
	}
	rand.Seed(Seed)
	// create chaos-mesh client
	log.Infof("Creating a chaos-mesh client")
	chaosClient, err := chaos_mesh.CreateClient(config.KubernetesNamespace, kubeClient)
	if err != nil {
		return err
	}

	var testArtifacts []*artifacts.TestArtifact
	var done = make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig, "Signal received. Ending after next test is completed.")
		done <- true // Signal that we're done
	}()

	for {
		select {
		case <-done:
			fmt.Println("Writing test artifacts")
			return cleanup(testArtifacts)
		default:
			clientUnderTest, isExec := pickRandomClient(config)
			targetSpec := getTargetSpec()
			attackSize := getRandomAttackSize()

			targetFilter, err := suite.TargetSpecEnumToLambda(targetSpec, isExec)
			if err != nil {
				return err
			}
			nodeFilter := suite.BuildNodeFilteringLambda(clientUnderTest, isExec)
			targetSelectors, err := suite.BuildChaosMeshTargetSelectors(len(nodes), testableNodes, attackSize, nodeFilter, targetFilter)
			if err != nil {
				log.Warn("unable to satisfy targeting constraint. skipping")
				continue
			}

			var targetingDescription string
			if targetSpec == suite.TargetMatchingNode {
				targetingDescription = fmt.Sprintf("Impacting the full node of targeted %s clients. Injecting into %s of the matching targets.", clientUnderTest, attackSize)
			} else {
				targetingDescription = fmt.Sprintf("Impacting the client of targeted %s clients. Injecting into %s of the matching targets.", clientUnderTest, attackSize)
			}

			test, err := buildRandomLatencyTest(
				targetingDescription,
				targetSelectors,
			)
			if err != nil {
				return err
			}
			log.Info("Running test")
			executor := test_executor.CreateTestExecutor(chaosClient, *test)

			err = executor.RunTestPlan(ctx)
			if err != nil {
				log.Errorf("Error while running test")
				return err
			} else {
				log.Infof("Test steps completed.")
			}

			log.Info("Starting health checks")
			podsUnderTest, err := executor.GetPodsUnderTest()
			if err != nil {
				return err
			}

			hc, err := health.BuildHealthChecker(kubeClient, podsUnderTest, test.HealthConfig)
			if err != nil {
				return err
			}
			results, err := hc.RunChecks(ctx)
			if err != nil {
				return err
			}
			testArtifact := artifacts.BuildTestArtifact(results, podsUnderTest, *test)
			testArtifacts = append(testArtifacts, testArtifact)
			if !testArtifact.TestPassed {
				log.Warn("Some health checks failed. Stopping test suite.")
				return cleanup(testArtifacts)
			}

			//time.Sleep(WaitBetweenTestsSecs * time.Second)
		}
	}

	//cleanup(testArtifacts)
	//return nil
}

func cleanup(testArtifacts []*artifacts.TestArtifact) error {
	err := artifacts.SerializeTestArtifacts(testArtifacts)
	if err != nil {
		return err
	}
	return nil
}
