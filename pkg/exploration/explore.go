package exploration

const WaitBetweenTestsSecs = 60
const Seed = 666

/*
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

func buildRandomLatencyTest(testIdx int, targetDescription string, targetSelectors []*suite.ChaosTargetSelector) (*types.SuiteTest, error) {
	minDelayMilliSeconds := 10
	maxDelayMilliSeconds := 1000
	minDurationSeconds := 10
	maxDurationSeconds := 300
	minJitterMilliseconds := 10
	maxJitterMilliseconds := 1000
	minCorrelation := 0
	maxCorrelation := 100

	grace := time.Second * 600
	duration := time.Second * time.Duration(rand.Intn(maxDurationSeconds-minDurationSeconds)+minDurationSeconds)
	delay := time.Millisecond * time.Duration(rand.Intn(maxDelayMilliSeconds-minDelayMilliSeconds)+minDelayMilliSeconds)
	jitter := time.Millisecond * time.Duration(rand.Intn(maxJitterMilliseconds-minJitterMilliseconds)+minJitterMilliseconds)
	correlation := rand.Intn(maxCorrelation-minCorrelation) + minCorrelation
	loc := time.FixedZone("GMT", 0)
	timefmt := time.Now().In(loc).Format(http.TimeFormat)
	description := fmt.Sprintf("Apply %s network latency for %s. Jitter: %s, correlation: %d against %d targets. %s. TestIdx: %d, TestTime: %d, %s", delay, duration, jitter, correlation, len(targetSelectors), targetDescription, testIdx, time.Now().Unix(), timefmt)
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

func buildRandomClockSkewTest(testIdx int, targetDescription string, targetSelectors []*suite.ChaosTargetSelector) (*types.SuiteTest, error) {
	minDelaySeconds := -900
	maxDelaySeconds := 900
	minDurationSeconds := 10
	maxDurationSeconds := 600

	grace := time.Second * 600
	delay := fmt.Sprintf("%ds", rand.Intn(maxDelaySeconds-minDelaySeconds)+minDelaySeconds)
	duration := fmt.Sprintf("%ds", rand.Intn(maxDurationSeconds-minDurationSeconds)+minDurationSeconds)

	loc := time.FixedZone("GMT", 0)
	timefmt := time.Now().In(loc).Format(http.TimeFormat)
	description := fmt.Sprintf("Apply %s clock skew for %s against %d targets. %s. TestIdx: %d, TestTime: %d, %s", delay, duration, len(targetSelectors), targetDescription, testIdx, time.Now().Unix(), timefmt)
	log.Info(description)
	return suite.ComposeNodeClockSkewTest(
		description,
		targetSelectors,
		delay,
		duration,
		&grace,
	)
}

func buildRandomTest(testIdx int, targetDescription string, targetSelectors []*suite.ChaosTargetSelector) (*types.SuiteTest, error) {
	testId := rand.Intn(2)
	if testId == 0 {
		return buildRandomLatencyTest(testIdx, targetDescription, targetSelectors)
	}
	if testId == 1 {
		return buildRandomClockSkewTest(testIdx, targetDescription, targetSelectors)
	}
	return nil, stacktrace.NewError("unknown test id")
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

func StartExploration(config *plan.PlannerConfig, suitecfg *types.ConfigParsed) error {
	// todo: big refactor
	ctx, cancelCtxFunc := context.WithCancel(context.Background())
	defer cancelCtxFunc()

	enclave, err := runtime.SetupEnclave(ctx, suitecfg)
	if err != nil {
		return err
	}
	_ = enclave

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

	for _, n := range nodes {
		log.Infof("%s", suite.ConvertToNodeIdTag(len(nodes), n, "execution"))
		log.Infof("%s", suite.ConvertToNodeIdTag(len(nodes), n, "consensus"))
	}

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
	var done = make(chan bool, 2)
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig, "Signal received. Ending after next test is completed.")
		done <- true // Signal that we're done

	}()
	killall := false
	testIdx := 1
	skipUntilTest := 29
	for {
		loc := time.FixedZone("GMT", 0)
		log.Infof("Start loop. GMT time: %s", time.Now().In(loc).Format(http.TimeFormat))
		select {
		case <-done:
			fmt.Println("Writing test artifacts")
			return cleanup(testArtifacts)
		default:
			if killall {
				fmt.Println("Writing test artifacts")
				return cleanup(testArtifacts)
			}
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

			for _, selector := range targetSelectors {
				for _, s := range selector.Selector {
					msg := "Hitting "
					for _, target := range s.Values {
						msg = fmt.Sprintf("%s %s,", msg, target)
					}
					log.Info(msg)
				}
			}
			log.Infof("time: %d", time.Now().Unix())

			var targetingDescription string
			if targetSpec == suite.TargetMatchingNode {
				targetingDescription = fmt.Sprintf("Impacting the full node of targeted %s clients. Injecting into %s of the matching targets.", clientUnderTest, attackSize)
			} else {
				targetingDescription = fmt.Sprintf("Impacting the client of targeted %s clients. Injecting into %s of the matching targets.", clientUnderTest, attackSize)
			}

			test, err := buildRandomTest(
				testIdx,
				targetingDescription,
				targetSelectors,
			)

			if err != nil {
				return err
			}

			if skipUntilTest != -1 {
				if testIdx < skipUntilTest {
					testIdx += 1
					continue
				}
			}

			testIdx += 1
			log.Info("Running test")
			executor := test_executor.CreateTestExecutor(chaosClient, *test)

			err = executor.RunTestPlan(ctx)
			if err != nil {
				log.Errorf("Error while running test")
				fmt.Println("Writing test artifacts")
				return cleanup(testArtifacts)
			} else {
				log.Infof("Test steps completed.")
			}

			log.Infof("Starting health checks at %s", time.Now().In(loc).Format(http.TimeFormat))
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

				fmt.Println("Writing test artifacts")
				err := cleanup(testArtifacts)
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

*/
