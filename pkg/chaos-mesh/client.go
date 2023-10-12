package chaos_mesh

import (
	"attacknet/cmd/pkg/kubernetes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/kurtosis-tech/stacktrace"
	apiextensionclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"reflect"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type ChaosClient struct {
	//kubeApiClient *apiextensionclientset.Clientset
	kubeApiClient  pkgclient.Client
	chaosNamespace string
}

func CreateClient(namespace string) (*ChaosClient, error) {
	scheme := runtime.NewScheme()
	err := api.AddToScheme(scheme)
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to add chaos-mesh v1alpha1 to scheme")
	}

	kubeConfig, _, err := kubernetes.CreateKubeClient()
	client, err := pkgclient.New(kubeConfig, pkgclient.Options{Scheme: scheme})
	if err != nil {
		return nil, stacktrace.Propagate(err, "unable to create a kubernetes client")
	}

	// todo: validate chaos-mesh is installed

	return &ChaosClient{client, namespace}, nil
}

func (c *ChaosClient) StartFault(ctx context.Context, faultSpec map[string]interface{}) (*FaultSession, error) {
	kind := faultSpec["kind"].(string)

	if chaosKind, ok := api.AllKinds()[kind]; ok {
		chaos := chaosKind.SpawnObject()

		faultName := fmt.Sprintf("fault-%d", time.Now().Unix())
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
		return &FaultSession{client: c, faultSpec: faultSpec, Name: faultName, faultKind: chaosKind}, nil

	} else {
		return nil, stacktrace.Propagate(errors.New("invalid fault kind"), "invalid fault kind: %s", kind)
	}
}

func Test(ctx context.Context) error {

	kubeConfig, _, err := kubernetes.CreateKubeClient()
	if err != nil {
		return stacktrace.Propagate(err, "aaaaaaa")
	}
	chaosClient, err := CreateClient("kt-ethereum")
	if err != nil {
		return stacktrace.Propagate(err, "aaaaaaa")
	}

	test := map[string]interface{}{
		"apiVersion": "chaos-mesh.org/v1alpha1",
		"kind":       "NetworkChaos",
		"metadata": map[string]interface{}{
			"name":      "example-myresource",
			"namespace": "chaos-mesh",
		},
		"spec": map[string]interface{}{
			"myvalue": "Hello, World!",
		},
	}

	_, err = chaosClient.StartFault(ctx, test)
	return err

	apiExtensionClient, err := apiextensionclientset.NewForConfig(kubeConfig)
	if err != nil {
		return stacktrace.Propagate(err, "aaaaaaa")
	}
	crdName := "networkchaos.chaos-mesh.org"

	crd, err := apiExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return stacktrace.Propagate(err, "bbbbb")
	}

	//dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal(err)
	}

	//c := &v1alpha1.NetworkChaos{}
	//gvr := schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1", Resource: "networkchaos"}

	client, err := pkgclient.New(kubeConfig, pkgclient.Options{})
	if err != nil {
		log.Fatal(err)
	}

	myresourceInstance := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "chaos-mesh.org/v1alpha1",
			"kind":       "networkchaos",
			"metadata": map[string]interface{}{
				"name":      "example-myresource",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"myvalue": "Hello, World!",
			},
		},
	}

	err = client.Create(ctx, myresourceInstance)
	if err != nil {
		log.Fatal(err)
	}

	_ = crd

	/*
		crds, err := apiExtensionClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		if err != nil {
			return stacktrace.Propagate(err, "bbbbb")
		}

		for _, crd := range crds.Items {
			fmt.Printf("Found CRD: %s\n", crd.Name)
		}
	*/

	return nil
}
