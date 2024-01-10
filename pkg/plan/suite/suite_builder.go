package suite

import (
	planTypes "attacknet/cmd/pkg/plan/types"
	"attacknet/cmd/pkg/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"time"
)

func buildNodeFilteringLambda(clientType string, isExecClient bool) TargetCriteriaFilter {
	if isExecClient {
		return createExecClientFilter(clientType)
	} else {
		return createConsensusClientFilter(clientType)
	}
}

func buildTestsForFaultType(
	faultType planTypes.FaultTypeEnum,
	config map[string]string,
	targetSelectors []*TargetSelector) (*types.SuiteTest, error) {

	switch faultType {
	case planTypes.FaultClockSkew:
		skew, ok := config["skew"]
		if !ok {
			return nil, stacktrace.NewError("missing skew field for clock skew fault")
		}
		duration, ok := config["duration"]
		if !ok {
			return nil, stacktrace.NewError("missing duration field for clock skew fault")
		}
		description := fmt.Sprintf("Apply %s clock skew for %s against %d targets", skew, duration, len(targetSelectors))
		return buildNodeClockSkewTest(description, targetSelectors, skew, duration)
	case planTypes.FaultContainerRestart:
		description := fmt.Sprintf("Restarting %d targets", len(targetSelectors))
		return buildNodeRestartTest(description, targetSelectors)
	}

	return nil, nil
}

func ComposeTestSuite(
	config planTypes.PlannerFaultConfiguration,
	isExecClient bool,
	nodes []*planTypes.Node) ([]types.SuiteTest, error) {

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
			targetSelectors, err := BuildTargetSelectors(nodes, attackSize, nodeFilter, targetFilter)
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

				test, err := buildTestsForFaultType(config.FaultType, faultConfig, targetSelectors)
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