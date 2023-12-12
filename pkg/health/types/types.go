package types

type GenericNetworkChecker interface {
	RunAllChecks() ([]*CheckResult, error)
}

type CheckResult struct {
	// think of a better struct later
	TestName    string
	PodsPassing []string
	PodsFailing []string
}
