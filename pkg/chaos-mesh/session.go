package chaos_mesh

import (
	"context"
	"fmt"
	api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type FaultPhase string

type FaultStatus string

const (
	Starting   FaultStatus = "Starting"
	InProgress FaultStatus = "In Progress"
	Stopping   FaultStatus = "Stopping"
	Completed  FaultStatus = "Completed"
	Error      FaultStatus = "Error"
)

// succeeded (inject worked, now back to normal)
// failure?
// time out?

type FaultSession struct {
	client              *ChaosClient
	faultKind           *api.ChaosKind
	faultSpec           map[string]interface{}
	Name                string
	podsFailingRecovery map[string]*api.Record
	TestStartTime       time.Time
	TestDuration        *time.Duration
	TestEndTime         time.Time
}

func NewFaultSession(ctx context.Context, client *ChaosClient, faultKind *api.ChaosKind, faultSpec map[string]interface{}, name string) (*FaultSession, error) {
	now := time.Now()

	partial := &FaultSession{
		client:              client,
		faultKind:           faultKind,
		faultSpec:           faultSpec,
		Name:                name,
		podsFailingRecovery: map[string]*api.Record{},
		TestStartTime:       now,
	}
	duration, err := partial.getDuration(ctx)
	if err != nil {
		return nil, err
	}
	partial.TestDuration = duration
	partial.TestEndTime = now.Add(*duration)
	return partial, nil
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

func (f *FaultSession) getDetailedStatus(ctx context.Context) ([]*api.Record, error) {
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
	records, ok := recordsVal.Interface().([]*api.Record)
	if !ok {
		return nil, stacktrace.NewError("unable to cast chaos experiment status")
	}
	return records, nil
}

func (f *FaultSession) checkForFailedRecovery(record *api.Record) (bool, []string) {
	messages := map[string]string{}
	for i := len(record.Events) - 1; i >= 0; i-- {
		if record.Events[i].Type == api.TypeFailed {
			msg := record.Events[i].Message
			if record.Events[i].Operation == api.Recover {
				if _, exists := f.podsFailingRecovery[msg]; !exists {
					messages[msg] = msg
					f.podsFailingRecovery[msg] = record
				}
			} else {
				log.Errorf("Did not expect operation to be apply for record with message %s", msg)
			}
		}
	}
	if len(messages) == 0 {
		return false, make([]string, 0)
	}

	distinctMessages := make([]string, 0, len(messages))
	for key := range messages {
		distinctMessages = append(distinctMessages, key)
	}
	return true, distinctMessages
}

// todo: we need a better way of monitoring fault injection status. There's a ton of statefulness represented in
// chaos-mesh that we're glancing over. Situations such as a pod crashing during a fault may produce unexpected behavior
// in this code as it currently stands.
func (f *FaultSession) GetStatus(ctx context.Context) (FaultStatus, error) {
	records, err := f.getDetailedStatus(ctx)
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

			failing, messages := f.checkForFailedRecovery(podRecord)
			if failing {
				log.Warn("One or more pods failed to recover from the fault and may have crashed:")
				for _, msg := range messages {
					log.Warnf("Error message: %s", msg)
				}
			}
		}
	}

	// todo: check if unrecovered pods are failing to recover ^^ up here PodRecord.Events[-1].Operation = "Recover", Type="Failed". Emit Message

	if podsNotInjected > 0 {
		return Starting, nil
	}
	if podsInjectedNotRecovered > 0 && podsInjectedAndRecovered == 0 {
		return InProgress, nil
	}
	if podsInjectedAndRecovered+len(f.podsFailingRecovery) == len(records) {
		return Completed, nil
	}
	if podsInjectedNotRecovered > 0 && podsInjectedAndRecovered > 0 {
		return Stopping, nil
	}
	// should be impossible to get here
	msg := fmt.Sprintf("invalid state, podsInjectedNotRecovered: %d, podsInjectedAndRecovered: %d", podsInjectedNotRecovered, podsInjectedAndRecovered)
	panic(msg)
}

func (f *FaultSession) getDuration(ctx context.Context) (*time.Duration, error) {
	resource, err := f.getKubeResource(ctx)
	if err != nil {
		return nil, err
	}

	durationVal := reflect.ValueOf(resource).Elem().FieldByName("Spec").FieldByName("Duration")
	durationStr, ok := durationVal.Interface().(*string)
	if !ok {
		return nil, stacktrace.NewError("unable to cast durationVal to string")
	}
	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		return nil, err
	}
	return &duration, err
}
