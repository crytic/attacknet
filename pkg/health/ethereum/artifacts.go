package ethereum

import (
	chaosMesh "attacknet/cmd/pkg/chaos-mesh"
	healthTypes "attacknet/cmd/pkg/health/types"
	"attacknet/cmd/pkg/types"
	"github.com/kurtosis-tech/stacktrace"
	"gopkg.in/yaml.v3"
)

func CreateEthereumArtifactSerializer() healthTypes.ArtifactSerializer {
	return &artifactSerializer{
		artifacts: []*testArtifact{},
	}
}

func (e artifactSerializer) AddHealthCheckResult(
	result interface{},
	podsUnderTest []*chaosMesh.PodUnderTest,
	test types.SuiteTest,
) error {
	castResult, ok := result.(*healthCheckResult)
	if !ok {
		return stacktrace.NewError("cannot cast health check result %s to healthCheckResult", result)
	}

	var containersTargeted []string
	for _, p := range podsUnderTest {
		containersTargeted = append(containersTargeted, p.GetName())
	}

	testPassed := castResult.AllChecksPassed()

	artifact := &testArtifact{
		test.TestName,
		containersTargeted,
		testPassed,
		castResult,
	}

	e.artifacts = append(e.artifacts, artifact)
	return nil
}

func (e artifactSerializer) SerializeArtifacts() ([]byte, error) {
	bs, err := yaml.Marshal(e.artifacts)
	if err != nil {
		return nil, stacktrace.Propagate(err, "could not marshal test artifacts")
	}

	return bs, nil
}
