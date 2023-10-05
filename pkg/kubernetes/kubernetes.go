package kubernetes

import (
	"github.com/kurtosis-tech/stacktrace"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

func CreateKubeClient() (*rest.Config, *kubernetes.Clientset, error) {
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
