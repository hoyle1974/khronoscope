package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/types"
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
	Visible  bool
	Node     node
	Parent   *renderNode
	Children []*renderNode
	Mark     string
}

func (r *renderNode) ShouldTraverse() bool {
	return r.Visible && r.Node.GetExpand()
}
func (r *renderNode) GetChildren() []misc.Node {
	b := make([]misc.Node, len(r.Children))
	for i := range r.Children {
		b[i] = r.Children[i]
	}
	return b
}

// The render tree represents a subset of the nodes we know about
// what will be shown to the user
func buildRenderTree(node node, parent *renderNode, matchSearch func(node) bool) *renderNode {
	if node == nil {
		return nil
	}

	renderNode := &renderNode{
		Visible: matchSearch(node),
		Node:    node,
		Parent:  parent,
	}

	if marked, ok := node.(types.Marked); ok {
		renderNode.Mark = marked.GetMark()
	}

	// Traverse all children of the current node
	//if node.ShouldTraverse() {
	for _, child := range node.GetChildren() {
		if tn, ok := child.(*treeNode); ok {
			if renderChild := buildRenderTree(tn, renderNode, matchSearch); renderChild != nil {
				renderNode.Children = append(renderNode.Children, renderChild)
			}
		}
		if tl, ok := child.(*treeLeaf); ok {
			if renderChild := buildRenderTree(tl, renderNode, matchSearch); renderChild != nil {
				renderNode.Children = append(renderNode.Children, renderChild)
			}
		}
	}
	//}

	return renderNode
}

func filterTree(root *renderNode) {
	if root == nil {
		return
	}

	nc := []*renderNode{}
	for _, child := range root.Children {
		if child.Visible {
			filterTree(child)
			nc = append(nc, child)
		}
	}
	root.Children = nc
}

func createRenderTree(model TreeModel, search string) (*renderNode, int) {
	// Before we render we want to traverse the model and add visual data to the nodes
	// This includes line number and visibility state
	renderNodeRoot := buildRenderTree(model.root, nil, func(node node) bool {
		if len(search) == 0 {
			return true
		}
		switch n := node.(type) {
		case *treeLeaf:
			return strings.Contains(n.Resource.String(), search)
		default:
			return false
		}
	})

	// If a child is filtered in the search then make the parents filtered
	if len(search) != 0 {
		renderNodeRoot.Visible = true
		misc.IterateTree(renderNodeRoot, func(n misc.Node) {
			rn := n.(*renderNode)
			temp := rn.Parent
			for temp != nil && rn.Visible {
				temp.Visible = true
				temp = temp.Parent
			}
			return
		})

		// Root is always visible
		renderNodeRoot.Visible = true // Ensure root is visible
		renderNodeRoot.Children[0].Visible = true
		renderNodeRoot.Children[1].Visible = true
		renderNodeRoot.Children[2].Visible = true

		filterTree(renderNodeRoot)
	}

	max := renumber(renderNodeRoot)

	return renderNodeRoot, max
}

func hasCollapsedParent(node *renderNode) bool {
	parent := node.Parent

	for parent != nil {
		if !parent.Node.GetExpand() {
			return true
		}
		parent = parent.Parent
	}

	return false
}

func renumber(renderNodeRoot *renderNode) int {
	lineNo := 0
	misc.TraverseNodeTree(renderNodeRoot, func(n misc.Node) bool {
		rn := n.(*renderNode)
		if rn.Visible && !hasCollapsedParent(rn) {
			rn.Line = lineNo
			lineNo++
		} else {
			rn.Line = -1
		}
		return false
	})
	return lineNo
}

type Logger interface {
	IsLogging(uid string) bool
}

func treeRender(renderNodeRoot *renderNode, cursorPos int, filter string, lc Logger) string {
	b := strings.Builder{}

	cms := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF22")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1111")).Bold(true)

	curLinePos := -1
	line := func(node *renderNode) string {
		l := " "
		if node != nil && node.Node != nil && lc.IsLogging(node.Node.GetUid()) {
			l = red.Render("◉")
		}

		curLinePos++
		if node != nil {
			if cursorPos == node.Line {
				return cms.Render("»") + l + " "
			}
			return " " + l + " "
		}
		return " "
	}

	namespaces := renderNodeRoot.Children[0]
	nodes := renderNodeRoot.Children[1]
	details := renderNodeRoot.Children[2]

	for _, node := range []*renderNode{namespaces, nodes} {
		if node.Node.GetExpand() {
			b.WriteString(line(node) + node.Node.GetTitle() + "\n")
			numOfChildren := len(node.Children)
			for idx, child := range node.Children {
				leaf := child.Node.(*treeLeaf)
				b.WriteString(line(child) + " " + grommet(idx == numOfChildren-1, false) + "── " + leaf.Resource.String() + " " + leaf.GetTitle() + "\n")
			}
		} else {
			b.WriteString(line(node) + node.Node.GetTitle() + " { ... }\n")
		}
		b.WriteString("\n")
	}

	if details.Node.GetExpand() {
		b.WriteString(line(details) + details.Node.GetTitle() + "\n\n")
		for _, namespaceNode := range details.Children {

			if namespaceNode.Node.GetExpand() {
				b.WriteString(line(namespaceNode) + namespaceNode.Node.GetTitle() + "\n")
				numOfNamespaces := len(namespaceNode.Children)
				for namespaceIdx, kindNode := range namespaceNode.Children {

					if kindNode.Node.GetExpand() {
						b.WriteString(line(kindNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindNode.Node.GetTitle() + "\n")
						numOfKinds := len(kindNode.Children)
						for kindIdx, resourceNode := range kindNode.Children {
							b.WriteString(line(resourceNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, true) + "   " + grommet(kindIdx == numOfKinds-1, false) + "──" + resourceNode.Node.(*treeLeaf).Resource.String() + "\n")
						}
					} else {
						b.WriteString(line(kindNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindNode.Node.GetTitle() + " { ... }\n")
					}
				}
			} else {
				b.WriteString(line(namespaceNode) + namespaceNode.Node.GetTitle() + " { ... }\n")
			}

		}
	} else {
		b.WriteString(line(details) + details.Node.GetTitle() + " { ... }\n")
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
		} else {
			filteredStrings = append(filteredStrings, str)
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
