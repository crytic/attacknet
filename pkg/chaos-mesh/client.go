package chaos_mesh

import (
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/kurtosis-tech/stacktrace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

type ChaosClient struct {
	kubeApiClient  pkgclient.Client
	chaosNamespace string
}

func CreateClient(namespace string, kubeClient *kubernetes.KubeClient) (*ChaosClient, error) {
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	chaosScheme := runtime.NewScheme()
	err := api.AddToScheme(chaosScheme)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to add chaos-mesh v1alpha1 to scheme")
	}

	err = corev1.AddToScheme(chaosScheme)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to add kubernetes core to scheme")
	}

	client, err := kubeClient.CreateDerivedClientWithSchema(chaosScheme)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create a kubernetes API client")
	}

	// todo: validate chaos-mesh is installed

	return &ChaosClient{client, namespace}, nil
}

func (c *ChaosClient) StartFault(ctx context.Context, faultSpec map[string]interface{}) (*FaultSession, error) {
	kindObj, exists := faultSpec["kind"]
	if !exists {
		return nil, stacktrace.NewError("unable to find 'kind' within fault spec")
	}

	kind, ok := kindObj.(string)
	if !ok {
		return nil, stacktrace.NewError("unable to cast faultSpec.Kind to string")
	}

	if chaosKind, ok := api.AllKinds()[kind]; ok {
		chaos := chaosKind.SpawnObject()

		faultName := fmt.Sprintf("fault-%d", time.Now().UnixMicro())
		faultMeta := metav1.ObjectMeta{Name: faultName, Namespace: c.chaosNamespace}

		reflect.ValueOf(chaos).Elem().FieldByName("ObjectMeta").Set(reflect.ValueOf(faultMeta))
		marshalled, err := json.Marshal(faultSpec)
		if err != nil {
			return nil, stacktrace.Propagate(err, "could not marshal faultspec")
		}

		err = json.Unmarshal(marshalled, &chaos)
		if err != nil {
			return nil, stacktrace.Propagate(err, "could not unmarshal faultspec")
		}

		err = c.kubeApiClient.Create(ctx, chaos)
		if err != nil {
			return nil, stacktrace.Propagate(err, "could not create custom resource")
		}

		return NewFaultSession(ctx, c, chaosKind, faultSpec, faultName)
	} else {
		return nil, stacktrace.Propagate(errors.New("invalid fault kind"), "invalid fault kind: %s", kind)
	}
}

func (c *ChaosClient) GetPodLabels(ctx context.Context, podName string) (map[string]string, error) {
	key := pkgclient.ObjectKey{
		Namespace: c.chaosNamespace,
		Name:      podName,
	}
	pod := &corev1.Pod{}

	err := c.kubeApiClient.Get(ctx, key, pod)
	if err != nil {
		return nil, err
	}
	labels := pod.GetLabels()

	return labels, nil
}
