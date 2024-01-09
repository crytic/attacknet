package pkg

import (
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/types"
	"context"
	"fmt"
	grafanaSdk "github.com/grafana-tools/sdk"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
)

// note: we may move the grafana logic to the health module if we move towards grafana-based health alerts
type GrafanaTunnel struct {
	Client                   *grafanaSdk.Client
	portForwardStopCh        chan struct{}
	allowPostFaultInspection bool
	cleanedUp                bool
}

func CreateGrafanaClient(ctx context.Context, kubeClient *kubernetes.KubeClient, config types.AttacknetConfig) (*GrafanaTunnel, error) {
	podName := config.GrafanaPodName
	exists, err := kubeClient.PodExists(ctx, podName)
	if err != nil {
		return nil, stacktrace.Propagate(err, "error while determining whether pod exists %s", podName)
	}
	if !exists {
		return nil, stacktrace.NewError("unable to locate grafana pod %s", podName)
	}

	var port uint16
	_, err = fmt.Sscan(config.GrafanaPodPort, &port)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to decode port number %s", config.GrafanaPodPort)
	}

	stopCh, err := kubeClient.StartPortForwarding(podName, int(port), int(port), true)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to start port forwarder")
	}

	client, err := grafanaSdk.NewClient("http://localhost:3000", "", grafanaSdk.DefaultHTTPClient)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create Grafana client")
	}

	return &GrafanaTunnel{client, stopCh, config.AllowPostFaultInspection, false}, nil
}

func (t *GrafanaTunnel) Cleanup(skipInspection bool) {
	if !t.cleanedUp {
		if t.allowPostFaultInspection && !skipInspection {
			log.Info("Press enter to terminate the port-forward connection.")
			_, _ = fmt.Scanln()
		}
		close(t.portForwardStopCh)
		t.cleanedUp = true
	}
}
