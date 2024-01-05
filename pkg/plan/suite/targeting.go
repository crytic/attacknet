package suite

import (
	"attacknet/cmd/pkg/plan/network"
	"fmt"
)

type TargetSelector struct {
	Selector    []ExpressionSelector
	Description string
}

type AttackSize string

const (
	TargetOne           AttackSize = "Target One"
	TargetAll           AttackSize = "Target all"
	TargetMinority      AttackSize = "Target Minority"           // scope will be 0<x<33.333
	TargetLiveness      AttackSize = "Target Liveness Threshold" // scope will be 33.333 < x < 50
	TargetMajority      AttackSize = "Target Majority"           // scope will be 50 < x < 66.6
	TargetSupermajority AttackSize = "Target Supermajority"      // scope will be 66.66 < x < 100
	// if we want to add 33.33%, 66.66%, and 50%, there needs to be logic to verify the network can be split into such
	// fractions, otherwise using the number in target selection may be misleading.
)

type CannotMeetConstraintError struct {
	AttackSize
	TargetableCount int
}

func (e CannotMeetConstraintError) Error() string {
	return fmt.Sprintf("Cannot target '%s' for %d nodes", e.AttackSize, e.TargetableCount)
}

type NodeFilterCriteria func(n *network.Node) bool

type TargetCriteriaFilter func(AttackSize, []*network.Node) ([]*network.Node, error)

type NodeImpactSelector func(node *network.Node) *TargetSelector

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
	case TargetOne:
		nodesToTarget = 1
	case TargetAll:
		nodesToTarget = len(targetable)
	case TargetMinority:
		nodesToTarget = int(totalTargetable * 0.32)
	case TargetLiveness:
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
	case TargetMajority:
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
	case TargetSupermajority:
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

func impactNode(node *network.Node) *TargetSelector {
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

func impactExecClient(node *network.Node) *TargetSelector {
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

func impactConsensusClient(node *network.Node) *TargetSelector {
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

func createExecClientTargetCriteria(elClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func createConsensusClientTargetCriteria(clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func createDualClientTargetCriteria(elClientType, clClientType string) TargetCriteriaFilter {
	return func(size AttackSize, nodes []*network.Node) ([]*network.Node, error) {
		criteria := func(n *network.Node) bool {
			return n.Consensus.Type == clClientType && n.Execution.Type == elClientType
		}
		targetableNodes := filterNodes(nodes, criteria)

		return chooseTargetsUsingAttackSize(size, targetableNodes)
	}
}

func BuildTargetSelectors(nodes []*network.Node, size AttackSize, targetCriteria TargetCriteriaFilter, impactSelector NodeImpactSelector) ([]*TargetSelector, error) {
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
