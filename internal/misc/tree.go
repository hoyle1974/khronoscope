package misc

type Node interface {
	IsLeaf() bool
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

	// If the current node is a parent, traverse its children
	if !node.IsLeaf() {
		// Traverse all children of the current node
		if node.ShouldTraverse() {
			for _, child := range node.GetChildren() {
				if foundNode := TraverseNodeTree(child, evaluator); foundNode != nil {
					return foundNode
				}
			}
		}
	}
	return nil
}
