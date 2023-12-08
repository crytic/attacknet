package pkg

import (
	"attacknet/cmd/pkg/kubernetes"
	"attacknet/cmd/pkg/project"
	"context"
	"fmt"
	grafanaSdk "github.com/grafana-tools/sdk"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GrafanaTunnel struct {
	Client                   *grafanaSdk.Client
	portForwardStopCh        chan struct{}
	allowPostFaultInspection bool
	cleanedUp                bool
}

func CreateGrafanaClient(ctx context.Context, namespace string, config project.AttacknetConfig) (*GrafanaTunnel, error) {
	kubeConfig, kubeClient, err := kubernetes.CreateKubeClient()
	if err != nil {
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, config.GrafanaPodName, metav1.GetOptions{})
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to locate grafana pod %s", config.GrafanaPodName)
	}
	var port uint16
	_, err = fmt.Sscan(config.GrafanaPodPort, &port)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to decode port number %s", config.GrafanaPodPort)
	}

	stopCh, err := kubernetes.StartPortForwarding(pod.Name, pod.Namespace, port, kubeConfig)
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
