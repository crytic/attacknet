package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"testing"
)

func NewMockNetworkUnconfigured(nodeCount int) []*network.Node {
	nodes := make([]*network.Node, nodeCount)
	for i := range nodes {
		nodes[i] = &network.Node{}
	}
	return nodes
}

func TestChooseTargetsUsingAttackSize(t *testing.T) {
	type testCase struct {
		NetworkSize           int
		TargetableCount       int
		AttackSize            AttackSize
		ExpectedResultSize    int
		ExpectConstraintError bool
	}

	testCases := []testCase{
		// test baseline cases where network size == targetable
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackAll,
			ExpectedResultSize: 10,
		},
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackOne,
			ExpectedResultSize: 1,
		},
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackMinority,
			ExpectedResultSize: 3,
		},
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackSuperminority,
			ExpectedResultSize: 4,
		},
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackMajority,
			ExpectedResultSize: 6,
		},
		{
			NetworkSize:        10,
			TargetableCount:    10,
			AttackSize:         AttackSupermajority,
			ExpectedResultSize: 7,
		},
		// test cases where network size != targetable
		{
			NetworkSize:        10,
			TargetableCount:    5,
			AttackSize:         AttackAll, // special case where not hitting 100% is ok
			ExpectedResultSize: 5,
		},
		{
			NetworkSize:        10,
			TargetableCount:    5,
			AttackSize:         AttackSuperminority,
			ExpectedResultSize: 4,
		},
		{
			NetworkSize:        10,
			TargetableCount:    5,
			AttackSize:         AttackMinority,
			ExpectedResultSize: 3,
		},

		// test a few cases that should produce constraint errors
		{
			NetworkSize:           10, // can only target up to 50%
			TargetableCount:       5,
			AttackSize:            AttackSupermajority,
			ExpectedResultSize:    -1,
			ExpectConstraintError: true,
		},
		{
			NetworkSize:           4,
			TargetableCount:       4, //can only target 50% or 75%
			AttackSize:            AttackMajority,
			ExpectedResultSize:    -1,
			ExpectConstraintError: true,
		},
	}

	for i, test := range testCases {
		nodes := NewMockNetworkUnconfigured(test.TargetableCount)
		targets, err := chooseTargetsUsingAttackSize(test.AttackSize, test.NetworkSize, nodes)

		if err != nil {
			if !test.ExpectConstraintError {
				t.Fatalf("Fail case %d, expected err==nil, case %v", i, test)
			} else {
				continue
			}
		}
		if len(targets) != test.ExpectedResultSize {
			t.Fatalf("Fail case %d, expected %d targets selected, received %d. case %v", i, test.ExpectedResultSize, len(targets), test)
		}

	}
}
