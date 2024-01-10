package suite

import (
	"attacknet/cmd/pkg/plan/types"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
)

type TargetSelector struct {
	Selector    []ExpressionSelector
	Description string
}

type CannotMeetConstraintError struct {
	types.AttackSize
	TargetableCount int
}

func (e CannotMeetConstraintError) Error() string {
	return fmt.Sprintf("Cannot target '%s' for %d nodes", e.AttackSize, e.TargetableCount)
}

type NodeFilterCriteria func(n *types.Node) bool

type TargetCriteriaFilter func(types.AttackSize, []*types.Node) ([]*types.Node, error)

type NodeImpactSelector func(node *types.Node) *TargetSelector

func filterNodes(nodes []*types.Node, criteria NodeFilterCriteria) []*types.Node {
	var result []*types.Node
	for _, n := range nodes {
		if criteria(n) {
			result = append(result, n)
		}
	}
	return result
}

func chooseTargetsUsingAttackSize(size types.AttackSize, targetable []*types.Node) ([]*types.Node, error) {
	totalTargetable := float32(len(targetable))
	var nodesToTarget int
	switch size {
	case types.AttackOne:
		nodesToTarget = 1
	case types.AttackAll:
		nodesToTarget = len(targetable)
	case types.AttackMinority:
		nodesToTarget = int(totalTargetable * 0.32)
	case types.AttackSuperminority:
		nodesToTarget = int(totalTargetable * 0.34)
		if float32(nodesToTarget)/totalTargetable < 0.333333333 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/totalTargetable >= 0.50 {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: len(targetable),
			}
		}
	case types.AttackMajority:
		nodesToTarget = int(totalTargetable * 0.51)
		if float32(nodesToTarget)/totalTargetable <= 0.50 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/totalTargetable >= 0.66 {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: len(targetable),
			}
		}
	case types.AttackSupermajority:
		nodesToTarget = int(totalTargetable * 0.67)
		if float32(nodesToTarget)/totalTargetable <= 0.666666666 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/totalTargetable > 1 {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: len(targetable),
			}
		}
	}

	if nodesToTarget == 0 {
		return nil, CannotMeetConstraintError{
			AttackSize:      size,
			TargetableCount: len(targetable),
		}
	}

	var targets []*types.Node
	for i := 0; i < nodesToTarget; i++ {
		targets = append(targets, targetable[i])
	}
	return targets, nil
}

func impactNode(node *types.Node) *TargetSelector {
	var targets []string

	elId := convertToNodeIdTag(node, Execution)
	targets = append(targets, elId)

	clId := convertToNodeIdTag(node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := convertToNodeIdTag(node, Validator)
		targets = append(targets, valId)
	}

	selector := ExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   targets,
	}

	description := fmt.Sprintf("%s/%s Node (Node #%d)", node.Execution.Type, node.Consensus.Type, node.Index)
	return &TargetSelector{
		Selector:    []ExpressionSelector{selector},
		Description: description,
	}
}

func impactExecClient(node *types.Node) *TargetSelector {
	elId := convertToNodeIdTag(node, Execution)
	selector := ExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   []string{elId},
	}

	description := fmt.Sprintf("%s client of %s/%s Node (Node #%d)", node.Execution.Type, node.Execution.Type, node.Consensus.Type, node.Index)
	return &TargetSelector{
		Selector:    []ExpressionSelector{selector},
		Description: description,
	}
}

func impactConsensusClient(node *types.Node) *TargetSelector {
	var targets []string
	clId := convertToNodeIdTag(node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := convertToNodeIdTag(node, Validator)
		targets = append(targets, valId)
	}

	selector := ExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   targets,
	}

	description := fmt.Sprintf("%s client of %s/%s Node (Node #%d)", node.Consensus.Type, node.Execution.Type, node.Consensus.Type, node.Index)
	return &TargetSelector{
		Selector:    []ExpressionSelector{selector},
		Description: description,
	}
}

func targetSpecEnumToLambda(targetSelector types.TargetingSpec, isExecClient bool) (func(node *types.Node) *TargetSelector, error) {
	if targetSelector == types.TargetMatchingNode {
		return impactNode, nil
	}
	if targetSelector == types.TargetMatchingClient {
		if isExecClient {
			return impactExecClient, nil
		} else {
			return impactConsensusClient, nil
		}
	}
	return nil, stacktrace.NewError("target selector %s not supported", targetSelector)
}

func createExecClientFilter(elClientType string) TargetCriteriaFilter {
	return func(size types.AttackSize, nodes []*types.Node) ([]*types.Node, error) {
		criteria := func(n *types.Node) bool {
			return n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func createConsensusClientFilter(clClientType string) TargetCriteriaFilter {
	return func(size types.AttackSize, nodes []*types.Node) ([]*types.Node, error) {
		criteria := func(n *types.Node) bool {
			return n.Consensus.Type == clClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func createDualClientTargetCriteria(elClientType, clClientType string) TargetCriteriaFilter {
	return func(size types.AttackSize, nodes []*types.Node) ([]*types.Node, error) {
		criteria := func(n *types.Node) bool {
			return n.Consensus.Type == clClientType && n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func BuildTargetSelectors(nodes []*types.Node, size types.AttackSize, targetCriteria TargetCriteriaFilter, impactSelector NodeImpactSelector) ([]*TargetSelector, error) {
	targets, err := targetCriteria(size, nodes)
	if err != nil {
		return nil, err
	}

	var targetSelectors []*TargetSelector
	for _, node := range targets {
		targetSelectors = append(targetSelectors, impactSelector(node))
	}
	return targetSelectors, nil
}
