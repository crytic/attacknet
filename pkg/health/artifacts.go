package health

import (
	"attacknet/cmd/pkg/health/ethereum"
	"attacknet/cmd/pkg/health/types"
	"github.com/kurtosis-tech/stacktrace"
)

func BuildArtifactSerializer(networkType string) (types.ArtifactSerializer, error) {
	switch networkType {
	case "ethereum":
		return ethereum.CreateEthereumArtifactSerializer(), nil
	default:
		return nil, stacktrace.NewError("no networkType %s supported in artifact serializer", networkType)
	}
}
