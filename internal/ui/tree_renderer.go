package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/misc"
)

func grommet(is bool, isVertical bool) string {
	if isVertical {
		if !is {
			return "│"
		}
		return " "
	}
	if !is {
		return "├"
	}
	return "└"
}

type renderNode struct {
	Line     int
	Filtered bool
	Node     node
	Children []*renderNode
}

func (r *renderNode) ShouldTraverse() bool {
	return !r.Filtered && r.Node.GetExpand()
}
func (r *renderNode) GetChildren() []misc.Node {
	b := make([]misc.Node, len(r.Children))
	for i := range r.Children {
		b[i] = r.Children[i]
	}
	return b
}

func buildRenderTree(node node, filter func(node) bool) *renderNode {
	if node == nil {
		return nil
	}

	renderNode := &renderNode{
		Filtered: filter(node),
		Node:     node,
	}

	// Traverse all children of the current node
	if node.ShouldTraverse() {
		for _, child := range node.GetChildren() {
			if tn, ok := child.(*treeNode); ok {
				if renderChild := buildRenderTree(tn, filter); renderChild != nil {
					renderNode.Children = append(renderNode.Children, renderChild)
				}
			}
			if tl, ok := child.(*treeLeaf); ok {
				if renderChild := buildRenderTree(tl, filter); renderChild != nil {
					renderNode.Children = append(renderNode.Children, renderChild)
				}
			}
		}
	}

	return renderNode
}

func CreateRenderTree(model TreeModel, filter string) *renderNode {
	// Before we render we want to traverse the model and add visual data to the nodes
	// This includes line number and filter state
	renderNodeRoot := buildRenderTree(model.root, func(node node) bool {
		switch n := node.(type) {
		case *treeLeaf:
			match := strings.Contains(n.Resource.String(), filter)
			if !match && n.Parent != nil {
				match = strings.Contains(n.Parent.(*treeNode).GetTitle(), filter)
				if !match && n.Parent.GetParent() != nil {
					match = strings.Contains(n.Parent.GetParent().(*treeNode).GetTitle(), filter)
				}
			}
			return match
		default:
			return false
		}
	})

	// Assign line numbers to nodes
	lineNo := 0
	misc.TraverseNodeTree(renderNodeRoot, func(n misc.Node) bool {
		rn := n.(*renderNode)
		if !rn.Filtered {
			rn.Line = lineNo
			lineNo++
		} else {
			rn.Line = -1
		}
		return false
	})

	return renderNodeRoot
}

func TreeRender(renderNodeRoot *renderNode, cursorPos int, filter string) string {
	b := strings.Builder{}

	curLinePos := -1
	line := func(node *renderNode) string {
		curLinePos++
		if node != nil {
			if cursorPos == node.Line {
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	namespaces := renderNodeRoot.Children[0]
	nodes := renderNodeRoot.Children[1]
	details := renderNodeRoot.Children[2]

	for _, node := range []*renderNode{namespaces, nodes} {
		b.WriteString(line(node) + node.Node.GetTitle() + "\n")
		if node.Node.GetExpand() {
			numOfChildren := len(node.Children)
			for idx, child := range node.Children {
				leaf := child.Node.(*treeLeaf)
				b.WriteString(line(child) + " " + grommet(idx == numOfChildren-1, false) + "── " + leaf.Resource.String() + " " + leaf.GetTitle() + "\n")
			}
		} else {
			b.WriteString(line(nil) + "   ...\n")
		}
		b.WriteString(line(nil) + "\n")
	}

	b.WriteString(line(details) + details.Node.GetTitle() + "\n")
	if details.Node.GetExpand() {
		for _, namespaceNode := range details.Children {

			if namespaceNode.Node.GetExpand() {
				b.WriteString(line(namespaceNode) + namespaceNode.Node.GetTitle() + "\n")
				numOfNamespaces := len(namespaceNode.Children)
				for namespaceIdx, kindNode := range namespaceNode.Children {
					// kindTreeNode := kindNode.(*treeNode)

					if kindNode.Node.GetExpand() {
						b.WriteString(line(kindNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindNode.Node.GetTitle() + "\n")
						numOfKinds := len(kindNode.Children)
						for kindIdx, resourceNode := range kindNode.Children {
							// resourceLeafNode := resourceNode.(*treeLeaf)
							b.WriteString(line(resourceNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, true) + "   " + grommet(kindIdx == numOfKinds-1, false) + "──" + resourceNode.Node.(*treeLeaf).Resource.String() + "\n")
						}
					} else {
						b.WriteString(line(kindNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindNode.Node.GetTitle() + " { ... }\n")
					}
				}
			} else {
				b.WriteString(line(namespaceNode) + namespaceNode.Node.GetTitle() + "{ ... }\n")
			}

		}
	} else {
		b.WriteString(line(nil) + "   ...\n")
	}
	b.WriteString(line(nil) + "\n")

	return strings.Join(filterAndBoldStrings(filter, strings.Split(b.String(), "\n")), "\n")
}

func filterAndBoldStrings(filter string, stringsToFilter []string) []string {
	if filter == "" {
		return stringsToFilter // Return original slice if filter is empty
	}

	var filteredStrings []string
	boldStyle := lipgloss.NewStyle().Bold(true)

	for _, str := range stringsToFilter {
		if strings.Contains(str, filter) {
			indices := findFilterIndices(str, filter)
			newStr := ""
			lastIndex := 0
			for _, indexPair := range indices {
				newStr += str[lastIndex:indexPair[0]]
				newStr += boldStyle.Render(str[indexPair[0]:indexPair[1]])
				lastIndex = indexPair[1]
			}
			if lastIndex < len(str) {
				newStr += str[lastIndex:]
			}

			filteredStrings = append(filteredStrings, newStr)
		}
	}
	return filteredStrings
}

func findFilterIndices(str, filter string) [][]int {
	var indices [][]int
	lowerStr := strings.ToLower(str)
	lowerFilter := strings.ToLower(filter)
	startIndex := 0
	for {
		index := strings.Index(lowerStr[startIndex:], lowerFilter)
		if index == -1 {
			break
		}
		absoluteIndex := startIndex + index
		indices = append(indices, []int{absoluteIndex, absoluteIndex + len(filter)})
		startIndex = absoluteIndex + len(filter)
	}
	return indices
}
