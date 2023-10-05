package pkg

import (
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateGrafanaClient(ctx context.Context, namespace string, config AttacknetConfig) (chan struct{}, error) {
	kubeConfig, kubeClient, err := kubernetes.CreateKubeClient()
	if err != nil {
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, config.GrafanaPodName, metav1.GetOptions{})
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to locate grafana pod %s", config.GrafanaPodName)
	}
	var port uint16
	_, err = fmt.Sscan(config.GrafanaPodPort, &port)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to decode port number %s", config.GrafanaPodPort)
	}

	stopCh, err := kubernetes.StartPortForwarding(pod.Name, pod.Namespace, port, kubeConfig)
	return stopCh, err
}
