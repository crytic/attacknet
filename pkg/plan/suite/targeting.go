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
type TargetCriteriaFilter func(AttackSize, int, []*network.Node) ([]*network.Node, error)
type NodeImpactSelector func(networkNodeCount int, node *network.Node) *ChaosTargetSelector

func BuildNodeFilteringLambda(clientType string, isExecClient bool) TargetCriteriaFilter {
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

func chooseTargetsUsingAttackSize(size AttackSize, networkSize int, targetable []*network.Node) ([]*network.Node, error) {
	networkSizeFloat := float32(networkSize)
	var nodesToTarget int
	switch size {
	case AttackOne:
		nodesToTarget = 1
	case AttackAll:
		nodesToTarget = len(targetable)
	case AttackMinority:
		nodesToTarget = int(networkSizeFloat * 0.32)
		if nodesToTarget > len(targetable) {
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: networkSize,
			}
		}
	case AttackSuperminority:
		nodesToTarget = int(networkSizeFloat * 0.34)
		if float32(nodesToTarget)/networkSizeFloat < 0.333333333 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/networkSizeFloat >= 0.50 || nodesToTarget > len(targetable) {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: networkSize,
			}
		}
	case AttackMajority:
		nodesToTarget = int(networkSizeFloat * 0.51)
		if float32(nodesToTarget)/networkSizeFloat <= 0.50 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/networkSizeFloat >= 0.66 || nodesToTarget > len(targetable) {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: networkSize,
			}
		}
	case AttackSupermajority:
		nodesToTarget = int(networkSizeFloat * 0.67)
		if float32(nodesToTarget)/networkSizeFloat <= 0.666666666 {
			nodesToTarget += 1
		}
		if float32(nodesToTarget)/networkSizeFloat > 1 || nodesToTarget > len(targetable) {
			// not enough nodes to use this attack size
			return nil, CannotMeetConstraintError{
				AttackSize:      size,
				TargetableCount: networkSize,
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

func createTargetSelectorForNode(networkNodeCount int, node *network.Node) *ChaosTargetSelector {
	var targets []string

	elId := ConvertToNodeIdTag(networkNodeCount, node, Execution)
	targets = append(targets, elId)

	clId := ConvertToNodeIdTag(networkNodeCount, node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := ConvertToNodeIdTag(networkNodeCount, node, Validator)
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

func createTargetSelectorForExecClient(networkNodeCount int, node *network.Node) *ChaosTargetSelector {
	elId := ConvertToNodeIdTag(networkNodeCount, node, Execution)
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

func createTargetSelectorForConsensusClient(networkNodeCount int, node *network.Node) *ChaosTargetSelector {
	var targets []string
	clId := ConvertToNodeIdTag(networkNodeCount, node, Consensus)
	targets = append(targets, clId)

	if node.Consensus.HasValidatorSidecar {
		valId := ConvertToNodeIdTag(networkNodeCount, node, Validator)
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

func TargetSpecEnumToLambda(targetSelector TargetingSpec, isExecClient bool) (func(networkNodeCount int, node *network.Node) *ChaosTargetSelector, error) {
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
	return func(size AttackSize, targetableSetSize int, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)
		if targetableNodes == nil {
			return nil, stacktrace.NewError("unable to satisfy targeting constraint")
		}

		return chooseTargetsUsingAttackSize(size, targetableSetSize, targetableNodes)
	}
}

func filterNodesByConsensusClient(clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, targetableSetSize int, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableSetSize, targetableNodes)
	}
}

func filterNodesByClientCombo(elClientType, clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, targetableSetSize int, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType && n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableSetSize, targetableNodes)
	}
}

func BuildChaosMeshTargetSelectors(networkNodeCount int, nodes []*network.Node, size AttackSize, targetCriteria TargetCriteriaFilter, impactSelector NodeImpactSelector) ([]*ChaosTargetSelector, error) {
	targets, err := targetCriteria(size, len(nodes)+1, nodes)
	if err != nil {
		return nil, err
	}

	var targetSelectors []*ChaosTargetSelector
	for _, node := range targets {
		targetSelectors = append(targetSelectors, impactSelector(networkNodeCount, node))
	}
	return targetSelectors, nil
}
