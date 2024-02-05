package kubernetes

import (
	"errors"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"io"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net"
	"net/http"
	"net/url"
	"time"
)

type PortForwardsSession struct {
	stopCh     chan struct{}
	Pod        KubePod
	TargetPort int
	LocalPort  int
}

func (session *PortForwardsSession) Close() {
	close(session.stopCh)
}

func (c *KubeClient) StartMultiPortForwardToLabeledPods(
	pods []KubePod,
	labelKey, labelValue string,
	targetPort int) ([]*PortForwardsSession, error) {
	var podsToForward []KubePod

	for _, pod := range pods {
		if pod.MatchesLabel(labelKey, labelValue) {
			podsToForward = append(podsToForward, pod)
		}
	}

	portForwardSessions, err := c.StartMultiPortForwards(podsToForward, targetPort)
	return portForwardSessions, err
}

func (c *KubeClient) StartMultiPortForwards(pods []KubePod, targetPort int) ([]*PortForwardsSession, error) {
	sessions := make([]*PortForwardsSession, len(pods))

	for i, pod := range pods {
		localPort, err := getFreeEphemeralPort()
		if err != nil {
			return nil, err
		}
		stopCh, err := c.StartPortForwarding(pod.GetName(), localPort, targetPort, false)
		if err != nil {
			return nil, err
		}

		sessions[i] = &PortForwardsSession{
			stopCh:     stopCh,
			Pod:        pod,
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

func (c *KubeClient) StartPortForwarding(pod string, localPort, remotePort int, printToStdout bool) (stopCh chan struct{}, err error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(c.clientInternal)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to create roundtripper")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", c.namespace, pod)
	serverURL, err := url.Parse(c.clientInternal.Host)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to decode kubeconfig.Host: %s", c.clientInternal.Host)
	}
	serverURL.Path = path

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)
	portFwd := fmt.Sprintf("%d:%d", localPort, remotePort)

	readyCh := make(chan struct{}, 1)
	stopCh = make(chan struct{}, 1)
	errLogger := io.Discard
	stdLogger := io.Discard
	if printToStdout {
		logger := log.New()
		errLogger = CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.ErrorLevel))
		stdLogger = CreatePrefixWriter("[port-forward] ", logger.WriterLevel(log.InfoLevel))
	}

	portForward, err := portforward.New(dialer, []string{portFwd}, stopCh, readyCh, stdLogger, errLogger)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create port forward dialer")
	}
	log.Debugf("Starting port-forward to pod/%s:%d", pod, remotePort)

	portForwardIssueCh := make(chan error, 1)
	defer close(portForwardIssueCh)

	retriesRem := 5
	/*go func(retriesRem int) {
		for retriesRem > 0 {
			if err = portForward.ForwardPorts(); err != nil {
				retriesRem--
				time.Sleep(200 * time.Millisecond)
				if retriesRem == 0 {
					portForwardIssueCh <- stacktrace.Propagate(err, "unable to start port forward session")
				}
			}
		}
	}(retries)*/

	for retriesRem > 0 {
		if err = portForward.ForwardPorts(); err != nil {
			retriesRem--
			time.Sleep(200 * time.Millisecond)
		} else {
			return stopCh, nil
		}
	}
	//portForwardIssueCh <- stacktrace.Propagate(err, "unable to start port forward session")
	return nil, errors.New("Failed to establish port forward")

	/*select {
	case <-readyCh:
		log.Debugf("Port-forward established to pod/%s:%d", pod, remotePort)
	case <-time.After(time.Minute):
		return nil, errors.New("timed out after waiting to establish port forward")
	case err = <-portForwardIssueCh:
		return nil, err
	}
	return stopCh, nil*/
}
