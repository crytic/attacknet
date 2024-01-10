package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
)

type ChaosTargetSelector struct {
	Selector    []ChaosExpressionSelector
	Description string
}

type CannotMeetConstraintError struct {
	AttackSize
	TargetableCount int
}

func (e CannotMeetConstraintError) Error() string {
	return fmt.Sprintf("Cannot target '%s' for %d nodes", e.AttackSize, e.TargetableCount)
}

type NodeFilterCriteria func(n *network.Node) bool
type TargetCriteriaFilter func(AttackSize, []*network.Node) ([]*network.Node, error)
type NodeImpactSelector func(node *network.Node) *ChaosTargetSelector

func buildNodeFilteringLambda(clientType string, isExecClient bool) TargetCriteriaFilter {
	if isExecClient {
		return filterNodesByExecClient(clientType)
	} else {
		return filterNodesByConsensusClient(clientType)
	}
}

func filterNodes(nodes []*network.Node, criteria NodeFilterCriteria) []*network.Node {
	var result []*network.Node
	for _, n := range nodes {
		if criteria(n) {
			result = append(result, n)
		}
	}
	return result
}

func chooseTargetsUsingAttackSize(size AttackSize, targetable []*network.Node) ([]*network.Node, error) {
	totalTargetable := float32(len(targetable))
	var nodesToTarget int
	switch size {
	case AttackOne:
		nodesToTarget = 1
	case AttackAll:
		nodesToTarget = len(targetable)
	case AttackMinority:
		nodesToTarget = int(totalTargetable * 0.32)
	case AttackSuperminority:
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
	case AttackMajority:
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
	case AttackSupermajority:
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

	var targets []*network.Node
	for i := 0; i < nodesToTarget; i++ {
		targets = append(targets, targetable[i])
	}
	return targets, nil
}

func createTargetSelectorForNode(node *network.Node) *ChaosTargetSelector {
	var targets []string

	elId := convertToNodeIdTag(node, Execution)
	targets = append(targets, elId)

	clId := convertToNodeIdTag(node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := convertToNodeIdTag(node, Validator)
		targets = append(targets, valId)
	}

	selector := ChaosExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   targets,
	}

	description := fmt.Sprintf("%s/%s Node (Node #%d)", node.Execution.Type, node.Consensus.Type, node.Index)
	return &ChaosTargetSelector{
		Selector:    []ChaosExpressionSelector{selector},
		Description: description,
	}
}

func createTargetSelectorForExecClient(node *network.Node) *ChaosTargetSelector {
	elId := convertToNodeIdTag(node, Execution)
	selector := ChaosExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   []string{elId},
	}

	description := fmt.Sprintf("%s client of %s/%s Node (Node #%d)", node.Execution.Type, node.Execution.Type, node.Consensus.Type, node.Index)
	return &ChaosTargetSelector{
		Selector:    []ChaosExpressionSelector{selector},
		Description: description,
	}
}

func createTargetSelectorForConsensusClient(node *network.Node) *ChaosTargetSelector {
	var targets []string
	clId := convertToNodeIdTag(node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := convertToNodeIdTag(node, Validator)
		targets = append(targets, valId)
	}

	selector := ChaosExpressionSelector{
		Key:      "kurtosistech.com/id",
		Operator: "In",
		Values:   targets,
	}

	description := fmt.Sprintf("%s client of %s/%s Node (Node #%d)", node.Consensus.Type, node.Execution.Type, node.Consensus.Type, node.Index)
	return &ChaosTargetSelector{
		Selector:    []ChaosExpressionSelector{selector},
		Description: description,
	}
}

func targetSpecEnumToLambda(targetSelector TargetingSpec, isExecClient bool) (func(node *network.Node) *ChaosTargetSelector, error) {
	if targetSelector == TargetMatchingNode {
		return createTargetSelectorForNode, nil
	}
	if targetSelector == TargetMatchingClient {
		if isExecClient {
			return createTargetSelectorForExecClient, nil
		} else {
			return createTargetSelectorForConsensusClient, nil
		}
	}
	return nil, stacktrace.NewError("target selector %s not supported", targetSelector)
}

func filterNodesByExecClient(elClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func filterNodesByConsensusClient(clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func filterNodesByClientCombo(elClientType, clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType && n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func buildChaosMeshTargetSelectors(nodes []*network.Node, size AttackSize, targetCriteria TargetCriteriaFilter, impactSelector NodeImpactSelector) ([]*ChaosTargetSelector, error) {
	targets, err := targetCriteria(size, nodes)
	if err != nil {
		return nil, err
	}

	var targetSelectors []*ChaosTargetSelector
	for _, node := range targets {
		targetSelectors = append(targetSelectors, impactSelector(node))
	}
	return targetSelectors, nil
}
