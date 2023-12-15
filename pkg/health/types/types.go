package types

import "context"

type GenericNetworkChecker interface {
	RunAllChecks(context.Context) ([]*CheckResult, error)
}

type CheckResult struct {
	// think of a better struct later
	TestName    string
	PodsPassing []string
	PodsFailing []string
}
