package kubernetes

import (
	"errors"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net"
	"net/http"
	"net/url"
	"time"
)

type PortForwardsSession struct {
	stopCh     chan struct{}
	PodName    string
	TargetPort int
	LocalPort  int
}

func (session *PortForwardsSession) Close() {
	close(session.stopCh)
}

func StartMultiPortForwards(pods []string, namespace string, targetPort int, kubeConfig *rest.Config) ([]*PortForwardsSession, error) {
	sessions := make([]*PortForwardsSession, len(pods))

	for i, podName := range pods {
		localPort, err := getFreeEphemeralPort()
		if err != nil {
			return nil, err
		}
		stopCh, err := StartPortForwarding(podName, namespace, localPort, targetPort, kubeConfig)
		if err != nil {
			return nil, err
		}

		sessions[i] = &PortForwardsSession{
			stopCh:     stopCh,
			PodName:    podName,
			TargetPort: targetPort,
			LocalPort:  localPort,
		}
	}
	return sessions, nil
}

// getFreeEphemeralPort note: you should use this port immediately otherwise another resource may claim it.
func getFreeEphemeralPort() (int, error) {

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, stacktrace.Propagate(err, "Error while finding new ephemeral port")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	if err != nil {
		return 0, stacktrace.Propagate(err, "Error while closing listener")
	}
	return port, nil
}

func StartPortForwarding(pod, namespace string, localPort, remotePort int, kubeConfig *rest.Config) (stopCh chan struct{}, err error) {
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
	portFwd := fmt.Sprintf("%d:%d", localPort, remotePort)

	stopCh = make(chan struct{}, 1)
	readyCh := make(chan struct{}, 1)
	logger := log.New()

	errLogger := CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.ErrorLevel))
	stdLogger := CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.InfoLevel))

	portForward, err := portforward.New(dialer, []string{portFwd}, stopCh, readyCh, stdLogger, errLogger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create port forward dialer")
	}
	log.Infof("Starting port-forward to pod/%s:%d", pod, remotePort)

	go func() {
		if err = portForward.ForwardPorts(); err != nil {
			panic(stacktrace.Propagate(err, "unable to start port forward session"))
		}
	}()

	select {
	case <-readyCh:
		log.Infof("Port-forward established to pod/%s:%d", pod, remotePort)
	case <-time.After(time.Minute):
		return nil, errors.New("timed out after waiting to establish port forward")
	}

	return stopCh, nil
}
