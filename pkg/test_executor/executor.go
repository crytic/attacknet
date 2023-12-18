package test_executor

import (
	chaos_mesh "attacknet/cmd/pkg/chaos-mesh"
	"attacknet/cmd/pkg/types"
	"context"
	"errors"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"time"
)

type TestExecutor struct {
	chaosClient   *chaos_mesh.ChaosClient
	testName      string
	planSteps     []types.PlanStep
	faultSessions []*chaos_mesh.FaultSession
}

func CreateTestExecutor(chaosClient *chaos_mesh.ChaosClient, test types.SuiteTest) *TestExecutor {
	return &TestExecutor{chaosClient: chaosClient, testName: test.TestName, planSteps: test.PlanSteps}
}

func (te *TestExecutor) RunTestPlan(ctx context.Context) error {
	for _, genericStep := range te.planSteps {
		marshalledSpec, err := yaml.Marshal(genericStep.Spec)
		if err != nil {
			return stacktrace.Propagate(err, "could not marshal plan step %s", genericStep.Spec)
		}
		log.Infof("Running test step: %s", genericStep.StepDescription)
		switch genericStep.StepType {
		case types.InjectFault:
			var s PlanStepSingleFault
			err = yaml.Unmarshal(marshalledSpec, &s)
			if err != nil {
				return stacktrace.Propagate(err, "could not unmarshal InjectFault from plan")
			}
			err = te.runInjectFaultStep(ctx, s) // check err after switch
		case types.WaitForFaultCompletion:
			var s PlanStepWaitForFaultCompletion
			err = yaml.Unmarshal(marshalledSpec, &s)
			if err != nil {
				return stacktrace.Propagate(err, "could not unmarshal WaitForFaultCompletion from plan")
			}
			err = te.runWaitForFaultCompletion(ctx, s)
		default:
			err = stacktrace.NewError("Unknown fault step type %s", genericStep.StepType)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (te *TestExecutor) runInjectFaultStep(ctx context.Context, step PlanStepSingleFault) error {
	faultSession, err := te.chaosClient.StartFault(ctx, step.FaultSpec)
	if err != nil {
		return err
	}

	err = waitForInjectionCompleted(ctx, faultSession)

	te.faultSessions = append(te.faultSessions, faultSession)
	/*
		var timeToSleep time.Duration
		if faultSession.TestDuration != nil {
			durationSeconds := int(faultSession.TestDuration.Seconds())
			log.Infof("Fault injected successfully. Fault will run for %d seconds before recovering.", durationSeconds)
			timeToSleep = *faultSession.TestDuration
		} else {
			log.Infof("Fault injected successfully. This fault has no specific duration.")
		}
		time.Sleep(timeToSleep)
	*/

	// we can build the health checker once the fault is injected
	/*
		log.Info("creating health checker")
		hc, err := health.BuildHealthChecker(cfg, kubeClient, faultSession.PodsUnderTest)
		if err != nil {
			return err
		}
		_ = hc
	*/

	// err = waitForFaultRecovery(ctx, faultSession)
	return err
}

func (te *TestExecutor) runWaitForFaultCompletion(ctx context.Context, _ PlanStepWaitForFaultCompletion) error {

	for i, fs := range te.faultSessions {
		now := time.Now()
		if now.Before(*fs.TestEndTime) {
			waitTime := fs.TestEndTime.Sub(now)
			log.Infof("Waiting %.0f seconds for fault #%d to terminate", waitTime.Seconds(), i+1)
			time.Sleep(waitTime)
		}
		err := waitForFaultRecovery(ctx, fs)
		if err != nil {
			return err
		}
		log.Infof("Fault #%d has completed", i+1)
	}
	return nil
}

func waitForInjectionCompleted(ctx context.Context, session *chaos_mesh.FaultSession) error {
	// First, wait 10 seconds to allow chaos-mesh to inject into the cluster.
	// If injection isn't complete after 10 seconds, something is  wrong and we should terminate.
	timeoutAt := time.Now().Add(time.Second * 10)

	for {
		if time.Now().After(timeoutAt) {
			errmsg := "chaos-mesh is still in a 'starting' state after 10 seconds. Check kubernetes events to see what's wrong."
			return stacktrace.NewError(errmsg)
		}

		status, err := session.GetStatus(ctx)
		if err != nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		switch status {
		case chaos_mesh.InProgress:
			log.Info("Fault injected successfully")
			return nil
		case chaos_mesh.Stopping:
			log.Warn("Fault changed to 'stopping' state immediately after injection. May indicate something is wrong.")
			return nil
		case chaos_mesh.Starting:
			if !session.TargetSelectionCompleted {
				errmsg := "chaos-mesh was unable to identify any pods for injection based on the configured criteria"
				return stacktrace.NewError(errmsg)
			}
		case chaos_mesh.Error:
			errmsg := "there was an unspecified error returned by chaos-mesh. inspect the fault resource"
			return stacktrace.NewError(errmsg)
		case chaos_mesh.Completed:
			// occurs for faults that perform an action immediately then terminate. (killing pods, etc)
			log.Info("Fault injected successfully")
			return nil
		default:
			return stacktrace.NewError("unknown chaos session state %s", status)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func waitForFaultRecovery(ctx context.Context, session *chaos_mesh.FaultSession) error {
	for {
		status, err := session.GetStatus(ctx)
		if err != nil {
			return err
		}

		switch status {
		case chaos_mesh.InProgress:
			log.Infof("The fault is still finishing up. Sleeping for 10s")
			time.Sleep(10 * time.Second)
		case chaos_mesh.Stopping:
			log.Infof("The fault is being stopped. Sleeping for 10s")
			time.Sleep(10 * time.Second)
		case chaos_mesh.Error:
			log.Errorf("there was an error returned by chaos-mesh")
			return errors.New("there was an unspecified error returned by chaos-mesh. inspect the fault resource")
		case chaos_mesh.Completed:
			log.Infof("The fault terminated successfully!")
			return nil
		default:
			return stacktrace.NewError("unknown chaos session state %s", status)
		}
		// todo: add timeout break if no changes in k8s resource after fault duration elapses
	}
}
