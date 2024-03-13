package kubernetes

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	//api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeClient struct {
	clientInternal *rest.Config
	clientset      *kubernetes.Clientset
	namespace      string
}

func CreateKubeClient(namespace string) (*KubeClient, error) {
	kubeConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to load the default kubeconfig file")
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to build a kubernetes client for the default types")
	}

	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create a kubernetes API client")
	}

	c := &KubeClient{
		clientInternal: kubeConfig,
		clientset:      kubeClient,
		namespace:      namespace,
	}

	return c, nil
}

func (c *KubeClient) CreateDerivedClientWithSchema(scheme *runtime.Scheme) (pkgclient.Client, error) {
	return pkgclient.New(c.clientInternal, pkgclient.Options{Scheme: scheme})
}

// todo: figure out the actual error conditions/pod doesnt exist error
func (c *KubeClient) PodExists(ctx context.Context, name string) (bool, error) {
	_, err := c.clientset.CoreV1().Pods(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return true, nil
	} else {
		return false, nil
	}
}

func (c *KubeClient) PodsMatchingLabel(ctx context.Context, labelKey, labelValue string) ([]KubePod, error) {
	selector := fmt.Sprintf("%s=%s", labelKey, labelValue)
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	var matchingPods []KubePod
	for _, pod := range pods.Items {
		labels := pod.GetLabels()
		matchingPods = append(matchingPods, &Pod{Name: pod.Name, Labels: labels})
	}

	return matchingPods, nil
}
