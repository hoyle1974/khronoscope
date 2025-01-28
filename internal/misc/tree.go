package misc

type Node interface {
	ShouldTraverse() bool
	GetChildren() []Node
}

func TraverseNodeTree(node Node, evaluator func(Node) bool) Node {
	if node == nil {
		return nil
	}

	// Evaluate the current node
	if evaluator(node) {
		return node
	}

	// Traverse all children of the current node
	if node.ShouldTraverse() {
		for _, child := range node.GetChildren() {
			if foundNode := TraverseNodeTree(child, evaluator); foundNode != nil {
				return foundNode
			}
		}
	}
	return nil
}
