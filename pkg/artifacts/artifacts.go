package artifacts

import (
	chaosMesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/health"
	healthTypes "attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/types"
	"errors"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	path2 "path"
	"time"
)

type TestArtifact struct {
	TestDescription    string                         `yaml:"test_description"`
	ContainersTargeted []string                       `yaml:"fault_injection_targets"`
	TestPassed         bool                           `yaml:"test_passed"`
	HealthResult       *healthTypes.HealthCheckResult `yaml:"health_check_results"`
}

func BuildTestArtifact(
	healthResults *healthTypes.HealthCheckResult,
	podsUnderTest []*chaosMesh.PodUnderTest,
	test types.SuiteTest,
) *TestArtifact {

	var containersTargeted []string
	for _, p := range podsUnderTest {
		containersTargeted = append(containersTargeted, p.GetName())
	}

	testPassed := health.AllChecksPassed(healthResults)

	return &TestArtifact{
		test.TestName,
		containersTargeted,
		testPassed,
		healthResults,
	}
}

func SerializeTestArtifacts(artifacts []*TestArtifact) error {
	artifactFilename := fmt.Sprintf("results-%d.yaml", time.Now().UnixMilli())

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := path2.Join(cwd, "artifacts")

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	artifactPath := path2.Join(path, artifactFilename)
	bs, err := yaml.Marshal(artifacts)
	if err != nil {
		return stacktrace.Propagate(err, "could not marshal test artifacts")
	}

	err = os.WriteFile(artifactPath, bs, 0600)
	if err != nil {
		return stacktrace.Propagate(err, "could not write artifacts to %s", artifactPath)
	}
	log.Infof("Wrote test artifact to %s", artifactPath)
	return nil
}
