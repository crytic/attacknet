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
	planCompleted bool
}

func CreateTestExecutor(chaosClient *chaos_mesh.ChaosClient, test types.SuiteTest) *TestExecutor {
	return &TestExecutor{chaosClient: chaosClient, testName: test.TestName, planSteps: test.PlanSteps}
}

func (te *TestExecutor) RunTestPlan(ctx context.Context) error {
	if te.planCompleted {
		return stacktrace.NewError("test executor %s has already been run", te.testName)
	}
	for i, genericStep := range te.planSteps {
		marshalledSpec, err := yaml.Marshal(genericStep.Spec)
		if err != nil {
			return stacktrace.Propagate(err, "could not marshal plan step %s", genericStep.Spec)
		}
		log.Infof("Running test step (%d/%d): '%s'", i+1, len(te.planSteps), genericStep.StepDescription)
		switch genericStep.StepType {
		case types.InjectFault:
			var s PlanStepSingleFault
			err = yaml.Unmarshal(marshalledSpec, &s)
			if err != nil {
				return stacktrace.Propagate(err, "could not unmarshal injectFault step from plan")
			}
			err = te.runInjectFaultStep(ctx, s) // check err after switch
		case types.WaitForFaultCompletion:
			var s PlanStepWaitForFaultCompletion
			err = yaml.Unmarshal(marshalledSpec, &s)
			if err != nil {
				return stacktrace.Propagate(err, "could not unmarshal waitForFaultCompletion step from plan")
			}
			err = te.runWaitForFaultCompletion(ctx, s)
		case types.WaitForDuration:
			var s PlanStepWait
			err = yaml.Unmarshal(marshalledSpec, &s)
			if err != nil {
				return stacktrace.Propagate(err, "could not unmarshal waitForDuration step from plan")
			}
			err = te.runWaitForDuration(s)
		default:
			err = stacktrace.NewError("Unknown fault step type %s", genericStep.StepType)
		}

		if err != nil {
			return err
		}
	}
	te.planCompleted = true
	return nil
}

func (te *TestExecutor) GetPodsUnderTest() ([]*chaos_mesh.PodUnderTest, error) {
	if !te.planCompleted {
		return nil, stacktrace.NewError("test %s has not been executed yet. cannot determine pods under test", te.testName)
	}
	pods := make(map[string]*chaos_mesh.PodUnderTest)
	var retPods []*chaos_mesh.PodUnderTest

	for _, session := range te.faultSessions {
		for _, pod := range session.PodsUnderTest {
			if val, ok := pods[pod.Name]; !ok {
				p := &chaos_mesh.PodUnderTest{
					Name:           pod.Name,
					Labels:         pod.Labels,
					ExpectDeath:    pod.ExpectDeath,
					TouchedByFault: pod.TouchedByFault,
				}
				pods[pod.Name] = p
				retPods = append(retPods, pod)
			} else {
				if pod.ExpectDeath && !val.ExpectDeath {
					val.ExpectDeath = true
				}
				if pod.TouchedByFault && !val.TouchedByFault {
					val.TouchedByFault = true
				}
			}
		}
	}
	
	return retPods, nil
}

func (te *TestExecutor) runInjectFaultStep(ctx context.Context, step PlanStepSingleFault) error {
	faultSession, err := te.chaosClient.StartFault(ctx, step.FaultSpec)
	if err != nil {
		return err
	}
	te.faultSessions = append(te.faultSessions, faultSession)

	err = waitForInjectionCompleted(ctx, faultSession)
	return err
}

func (te *TestExecutor) runWaitForFaultCompletion(ctx context.Context, _ PlanStepWaitForFaultCompletion) error {

	for i, fs := range te.faultSessions {
		now := time.Now()
		if now.Before(*fs.TestEndTime) {
			waitTime := fs.TestEndTime.Sub(now)
			log.Infof("Waiting %.0f seconds for fault #%d to terminate", waitTime.Seconds(), i+1)
			log.Infof(
				"Est time of fault completion: %d:%d:%d %s",
				fs.TestEndTime.Hour(),
				fs.TestEndTime.Minute(),
				fs.TestEndTime.Second(),
				fs.TestEndTime.Location().String())
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

func (te *TestExecutor) runWaitForDuration(step PlanStepWait) error {
	log.Infof("Sleeping for %.0f seconds", step.WaitAmount.Seconds())
	time.Sleep(step.WaitAmount)
	return nil
}

func waitForInjectionCompleted(ctx context.Context, session *chaos_mesh.FaultSession) error {
	// First, wait 10 seconds to allow chaos-mesh to inject into the cluster.
	// If injection isn't complete after 10 seconds, something is  wrong and we should terminate.
	timeoutAt := time.Now().Add(time.Second * 10)

	targetingGracePeriod := time.Now().Add(time.Second * 5)

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
				if time.Now().After(targetingGracePeriod) {
					errmsg := "chaos-mesh was unable to identify any pods for injection based on the configured criteria"
					return stacktrace.NewError(errmsg)
				}
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
	}
}
