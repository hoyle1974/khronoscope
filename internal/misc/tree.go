package misc

type Node interface {
	ShouldTraverse() bool
	GetChildren() []Node
}

func IterateTree(node Node, work func(Node)) {
	if node == nil {
		return
	}

	// Evaluate the current node
	work(node)

	// Traverse all children of the current node
	for _, child := range node.GetChildren() {
		IterateTree(child, work)
	}
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
