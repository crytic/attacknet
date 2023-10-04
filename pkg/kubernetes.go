package pkg

import (
	"context"
	"errors"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func createKubeClient() (*rest.Config, *kubernetes.Clientset, error) {
	kubeConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, nil, stacktrace.Propagate(err, "Unable to load the default kubeconfig file")
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, stacktrace.Propagate(err, "Unable to build a kubernetes client for the default config")
	}
	return kubeConfig, kubeClient, nil
}

func startPortForwarding(pod, namespace string, port uint16, kubeConfig *rest.Config) (stopCh chan struct{}, err error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(kubeConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to create roundtripper")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, pod)
	serverURL, err := url.Parse(kubeConfig.Host)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to decode kubeconfig.Host: %s", kubeConfig.Host)
	}
	serverURL.Path = path

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)
	portFwd := fmt.Sprintf("%d:%d", port, port)

	stopCh = make(chan struct{}, 1)
	readyCh := make(chan struct{}, 1)
	portForward, err := portforward.New(dialer, []string{portFwd}, stopCh, readyCh, os.Stdout, os.Stderr)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create port forward dialer")
	}
	fmt.Print("Starting port-forward to grafana pod")

	go func() {
		if err = portForward.ForwardPorts(); err != nil {
			panic(stacktrace.Propagate(err, "unable to start port forward session"))
			return
		}
	}()

	select {
	case <-readyCh:
		fmt.Print("Port-forward established.")
	case <-time.After(time.Minute):
		return nil, errors.New("timed out after waiting to establish port forward")
	}

	return stopCh, nil
}

func createGrafanaClient(ctx context.Context, namespace string, config AttacknetConfig) (chan struct{}, error) {
	kubeConfig, kubeClient, err := createKubeClient()
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

	stopCh, err := startPortForwarding(pod.Name, pod.Namespace, port, kubeConfig)
	return stopCh, err
}
