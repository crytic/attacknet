package chaos_mesh

import (
	"context"
	"fmt"
	api "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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

var FaultHasNoDurationErr = fmt.Errorf("this fault has no expected duration")

type FaultSession struct {
	client                   *ChaosClient
	faultKind                *api.ChaosKind
	faultType                string
	faultAction              string
	faultSpec                map[string]interface{}
	Name                     string
	podsFailingRecovery      map[string]*api.Record
	checkedForMissingPods    bool
	podsExpectedMissing      int
	TestStartTime            time.Time
	TestDuration             *time.Duration
	TestEndTime              *time.Time
	TargetSelectionCompleted bool
	PodsUnderTest            []*PodUnderTest
}

func NewFaultSession(ctx context.Context, client *ChaosClient, faultKind *api.ChaosKind, faultSpec map[string]interface{}, name string) (*FaultSession, error) {
	now := time.Now()

	faultKindStr, ok := faultSpec["kind"].(string)
	if !ok {
		return nil, stacktrace.NewError("failed to decode faultSpec.kind to string: %s", faultSpec["kind"])
	}

	spec, ok := faultSpec["spec"].(map[string]interface{})
	if !ok {
		return nil, stacktrace.NewError("failed to decode faultSpec.spec to map[string]interface{}")
	}

	faultAction, ok := spec["action"].(string)
	if !ok {
		faultAction = "default"
	}

	partial := &FaultSession{
		client:                   client,
		faultKind:                faultKind,
		faultType:                faultKindStr,
		faultSpec:                spec,
		faultAction:              faultAction,
		Name:                     name,
		podsFailingRecovery:      map[string]*api.Record{},
		TestStartTime:            now,
		podsExpectedMissing:      0,
		checkedForMissingPods:    false,
		TargetSelectionCompleted: false,
		PodsUnderTest:            nil,
	}
	duration, err := partial.getDuration(ctx)
	if err != nil {
		if err == FaultHasNoDurationErr {
			partial.TestDuration = nil
			partial.TestEndTime = nil
		} else {
			return nil, err
		}
	} else {
		partial.TestDuration = duration
		endTime := now.Add(*duration)
		partial.TestEndTime = &endTime
	}

	return partial, nil
}

func (f *FaultSession) getKubeFaultResource(ctx context.Context) (client.Object, error) {
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

// returns True if TargetSelectionCompleted becomes true
func (f *FaultSession) checkTargetSelectionCompleted(resource client.Object) (bool, error) {
	if f.TargetSelectionCompleted {
		return false, nil
	}
	conditionsVal := reflect.ValueOf(resource).Elem().FieldByName("Status").FieldByName("ChaosStatus").FieldByName("Conditions")
	conditions, ok := conditionsVal.Interface().([]api.ChaosCondition)
	if !ok || conditions == nil {
		return false, stacktrace.NewError("Unable to decode status.chaosstatus.conditions")
	}
	for _, condition := range conditions {
		if condition.Type != api.ConditionSelected {
			continue
		}
		if condition.Status == v1.ConditionTrue {
			f.TargetSelectionCompleted = true
		}

		return true, nil
	}
	return false, nil
}

func (f *FaultSession) getFaultRecords(ctx context.Context) ([]*api.Record, error) {
	resource, err := f.getKubeFaultResource(ctx)
	if err != nil {
		return nil, err
	}

	// note: we may be able to move this somewhere else once we get a better idea of fault lifecycle management.
	targetsSelected, err := f.checkTargetSelectionCompleted(resource)
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

	if targetsSelected && records != nil {
		err = f.populatePodsUnderTest(ctx, records)
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

// todo: check which pods are actually expected to die instead of the number of pods.
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

/*
This must be run after the fault manifest has been applied and the handler webhook has run.
*/
func (f *FaultSession) populatePodsUnderTest(ctx context.Context, records []*api.Record) error {
	if !f.checkedForMissingPods {
		f.checkedForMissingPods = true
		// we expect missing pods when the fault is pod kill.

		podsInjected, err := filterInjectedPods(records)
		if err != nil {
			return err
		}
		log.Infof("Chaos-mesh has identified %d pods matching the targeting criteria", len(podsInjected))
		if f.faultType == "PodChaos" && f.faultAction == "pod-kill" {
			f.podsExpectedMissing = len(podsInjected)
			log.Infof("We're expecting %d pods to be terminated from the selected fault", f.podsExpectedMissing)
		}

		// populate f.PodsUnderTest
		var podsUnderTest []*PodUnderTest
		if f.podsExpectedMissing > 0 {
			podsUnderTest, err = buildPodsUnderTestSlice(ctx, f.client, podsInjected, true)
		} else {
			podsUnderTest, err = buildPodsUnderTestSlice(ctx, f.client, podsInjected, false)
		}
		if err != nil {
			return err
		}
		f.PodsUnderTest = podsUnderTest

	} else {
		// suspect unreachable
		log.Error("this code is supposed to be unreachable")
	}
	return nil
}

// todo: we need a better way of monitoring fault injection status. There's a ton of statefulness represented in
// chaos-mesh that we're glancing over. Situations such as a pod crashing during a fault may produce unexpected behavior
// in this code as it currently stands.
func (f *FaultSession) GetStatus(ctx context.Context) (FaultStatus, error) {
	records, err := f.getFaultRecords(ctx)
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

	if podsNotInjected > 0 {
		return Starting, nil
	}
	if podsInjectedNotRecovered-f.podsExpectedMissing > 0 && podsInjectedAndRecovered == 0 {
		return InProgress, nil
	}
	if podsInjectedAndRecovered+len(f.podsFailingRecovery)+f.podsExpectedMissing == len(records) {
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
	resource, err := f.getKubeFaultResource(ctx)
	if err != nil {
		return nil, err
	}

	durationVal := reflect.ValueOf(resource).Elem().FieldByName("Spec").FieldByName("Duration")
	durationStr, ok := durationVal.Interface().(*string)
	if !ok {
		return nil, stacktrace.NewError("unable to cast durationVal to string")
	}
	if durationStr == nil {
		return nil, FaultHasNoDurationErr
	}

	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		return nil, err
	}
	return &duration, err
}

func buildPodsUnderTestSlice(ctx context.Context, client *ChaosClient, podNames []string, expectDeath bool) ([]*PodUnderTest, error) {
	podsUnderTest := make([]*PodUnderTest, len(podNames))

	for i, podName := range podNames {
		labels, err := client.GetPodLabels(ctx, podName)
		if err != nil {
			return nil, err
		}
		if labels == nil {
			return nil, stacktrace.NewError("pod %s had no labels", podName)
		}

		podsUnderTest[i] = &PodUnderTest{
			Name:        podName,
			Labels:      labels,
			ExpectDeath: expectDeath,
		}
	}
	return podsUnderTest, nil
}

// filterInjectedPods takes a list of chaos mesh records and returns a list of pod names that are currently in
// the injected phase.
func filterInjectedPods(records []*api.Record) ([]string, error) {
	var injectedPods []string
	for _, record := range records {
		if record.Phase == "Injected" {
			parts := strings.Split(record.Id, "/")
			if len(parts) <= 2 {
				return nil, stacktrace.NewError("fault record id was split into less than two parts")
			}
			injectedPods = append(injectedPods, parts[1])
		}
	}
	return injectedPods, nil
}
