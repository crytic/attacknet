package chaos_mesh

import (
	"context"
	"fmt"
	api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/kurtosis-tech/stacktrace"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type FaultPhase string

type FaultStatus string

const (
	Starting   FaultStatus = "Starting"
	InProgress             = "In Progress"
	Stopping               = "Stopping"
	Completed              = "Completed"
	Error                  = "Error"
)

// succeeded (inject worked, now back to normal)
// failure?
// time out?

type FaultSession struct {
	client    *ChaosClient
	faultKind *api.ChaosKind
	faultSpec map[string]interface{}
	Name      string
}

func (f *FaultSession) getKubeResource(ctx context.Context) (client.Object, error) {
	key := client.ObjectKey{
		Namespace: f.client.chaosNamespace,
		Name:      f.Name,
	}

	resource := f.faultKind.SpawnObject()
	err := f.client.kubeApiClient.Get(ctx, key, resource)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Unable to query for resource %s in namespace %s", key.Name, key.Namespace)
	}
	return resource, nil
}

func (f *FaultSession) GetDetailedStatus(ctx context.Context) ([]*api.Record, error) {
	resource, err := f.getKubeResource(ctx)
	if err != nil {
		return nil, err
	}

	// Feel free to figure out a better way to do this. These fields are part of every Chaos status struct we support,
	// but since they don't implement a common interface containing the status fields, there's no clean or simple way
	// to extract the values in Go. One alternate option may be to serialize to json, then deserialize into an object
	// that will ignore the fault-specific fields.
	// This section also can't handle errors. The code will panic if the resource isn't compliant, which isn't great.
	recordsVal := reflect.ValueOf(resource).Elem().FieldByName("Status").FieldByName("ChaosStatus").FieldByName("Experiment").FieldByName("Records")
	records := recordsVal.Interface().([]*api.Record)
	return records, nil
}

// todo: we need a better way of monitoring fault injection status. There's a ton of statefulness represented in
// chaos-mesh that we're glancing over. Situations such as a pod crashing during a fault may produce unexpected behavior
// in this code as it currently stands.
func (f *FaultSession) GetStatus(ctx context.Context) (FaultStatus, error) {
	records, err := f.GetDetailedStatus(ctx)
	if err != nil {
		return Error, err
	}

	if records == nil {
		return Starting, nil
	}

	podsInjectedAndRecovered := 0
	podsInjectedNotRecovered := 0
	podsNotInjected := 0

	for _, podRecord := range records {
		if podRecord.InjectedCount == 0 {
			podsNotInjected += 1
		} else if podRecord.InjectedCount == podRecord.RecoveredCount {
			podsInjectedAndRecovered += 1
		} else {
			podsInjectedNotRecovered += 1
		}
	}

	if podsNotInjected > 0 {
		return Starting, nil
	}
	if podsInjectedNotRecovered > 0 && podsInjectedAndRecovered == 0 {
		return InProgress, nil
	}
	if podsInjectedNotRecovered > 0 && podsInjectedAndRecovered > 0 {
		return Stopping, nil
	}
	if podsInjectedAndRecovered == len(records) {
		return Completed, nil
	}
	// should be impossible to get here
	msg := fmt.Sprintf("invalid state, podsInjectedNotRecovered: %s, podsInjectedAndRecovered: %s", podsInjectedNotRecovered, podsInjectedAndRecovered)
	panic(msg)
	return Error, nil
}

// todo: memoize or enshrine
func (f *FaultSession) GetDuration(ctx context.Context) (*time.Duration, error) {
	resource, err := f.getKubeResource(ctx)
	if err != nil {
		return nil, err
	}

	durationVal := reflect.ValueOf(resource).Elem().FieldByName("Spec").FieldByName("Duration")
	durationStr := durationVal.Interface().(*string)
	duration, err := time.ParseDuration(*durationStr)
	return &duration, err
}
