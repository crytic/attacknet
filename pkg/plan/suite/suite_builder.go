package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"attacknet/cmd/pkg/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

func ComposeTestSuite(
	config PlannerFaultConfiguration,
	isExecClient bool,
	nodes []*network.Node) ([]types.SuiteTest, error) {

	var tests []types.SuiteTest
	runtimeEstimate := 0

	nodeFilter := buildNodeFilteringLambda(config.TargetClient, isExecClient)

	for _, targetDimension := range config.TargetingDimensions {
		targetFilter, err := targetSpecEnumToLambda(targetDimension, isExecClient)
		if err != nil {
			return nil, err
		}
		nodeCountsTested := make(map[int]bool)
		for _, attackSize := range config.AttackSizeDimensions {
			targetSelectors, err := buildChaosMeshTargetSelectors(nodes, attackSize, nodeFilter, targetFilter)
			if err != nil {
				cannotMeet, ok := err.(CannotMeetConstraintError)
				if !ok {
					return nil, err
				}
				log.Infof("Attack size %s for %d nodes cannot be satisfied. Use more clients if this attack size needs to be tested.", cannotMeet.AttackSize, cannotMeet.TargetableCount)
				continue
			}
			// deduplicate attack sizes that produce the same scope
			_, alreadyTested := nodeCountsTested[len(targetSelectors)]
			if alreadyTested {
				continue
			} else {
				nodeCountsTested[len(targetSelectors)] = true
			}

			for _, faultConfig := range config.FaultConfigDimensions {
				// update runtime estimate. find better way
				duration, ok := faultConfig["duration"]
				if ok {
					d, err := time.ParseDuration(duration)
					if err == nil {
						runtimeEstimate += int(d.Seconds())
					}
				}
				var targetingDescription string
				if targetDimension == TargetMatchingNode {
					targetingDescription = fmt.Sprintf("Impacting the full node of targeted %s clients. Injecting into %s of the matching targets.", config.TargetClient, attackSize)
				} else {
					targetingDescription = fmt.Sprintf("Impacting the client of targeted %s clients. Injecting into %s of the matching targets.", config.TargetClient, attackSize)
				}

				test, err := composeTestForFaultType(
					config.FaultType,
					faultConfig,
					targetSelectors,
					targetingDescription,
				)
				if err != nil {
					return nil, err
				}
				tests = append(tests, *test)
			}
		}
	}

	log.Infof("ESTIMATE: Running this test suite will take, at minimum, %d minutes", runtimeEstimate/60)

	return tests, nil
}

func composeTestForFaultType(
	faultType FaultTypeEnum,
	config map[string]string,
	targetSelectors []*ChaosTargetSelector,
	targetingDescription string,
) (*types.SuiteTest, error) {

	switch faultType {
	case FaultClockSkew:
		skew, ok := config["skew"]
		if !ok {
			return nil, stacktrace.NewError("missing skew field for clock skew fault")
		}
		duration, ok := config["duration"]
		if !ok {
			return nil, stacktrace.NewError("missing duration field for clock skew fault")
		}
		grace, ok := config["grace_period"]
		if !ok {
			return nil, stacktrace.NewError("missing grace_period field for clock skew fault")
		}
		graceDuration, err := time.ParseDuration(grace)
		if err != nil {
			return nil, stacktrace.NewError("unable to convert grace_period field to a time duration for clock skew fault")
		}

		description := fmt.Sprintf("Apply %s clock skew for %s against %d targets. %s", skew, duration, len(targetSelectors), targetingDescription)
		return composeNodeClockSkewTest(description, targetSelectors, skew, duration, graceDuration)
	case FaultContainerRestart:
		grace, ok := config["grace_period"]
		if !ok {
			return nil, stacktrace.NewError("missing grace_period field for restsrt fault")
		}
		graceDuration, err := time.ParseDuration(grace)
		if err != nil {
			return nil, stacktrace.NewError("unable to convert grace_period field to a time duration for clock skew fault")
		}
		description := fmt.Sprintf("Restarting %d targets. %s", len(targetSelectors), targetingDescription)
		return composeNodeRestartTest(description, targetSelectors, graceDuration)
	}

	return nil, nil
}
