package suite

import "attacknet/cmd/pkg/plan/network"

type NodeSelector struct {
	Node     *network.Node
	Selector ExpressionSelector
}
