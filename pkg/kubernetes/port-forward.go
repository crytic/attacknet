package kubernetes

import (
	"errors"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"time"
)

func StartPortForwarding(pod, namespace string, port uint16, kubeConfig *rest.Config) (stopCh chan struct{}, err error) {
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
	logger := log.New()

	errLogger := CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.ErrorLevel))
	stdLogger := CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.InfoLevel))

	portForward, err := portforward.New(dialer, []string{portFwd}, stopCh, readyCh, stdLogger, errLogger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create port forward dialer")
	}
	log.Info("Starting port-forward to grafana pod")

	go func() {
		if err = portForward.ForwardPorts(); err != nil {
			panic(stacktrace.Propagate(err, "unable to start port forward session"))
		}
	}()

	select {
	case <-readyCh:
		log.Info("Port-forward established.")
	case <-time.After(time.Minute):
		return nil, errors.New("timed out after waiting to establish port forward")
	}

	return stopCh, nil
}
