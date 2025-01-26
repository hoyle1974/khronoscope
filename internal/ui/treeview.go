package ui

import (
	"maps"
	"slices"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/types"
)

// TreeView provides a way to browse a set of k8s resources in a tree view.
// It builds a view consisting of 3 sections: namespaces, nodes, and details.
// It manages cursor movement in the view, collapsing/expanding nodes and tries
// to keep the cursor mostly sane even when resources the cursor is on disappear.

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

type treeViewCursor struct {
	Pos  int
	Uid  string
	Node node
}

type node interface {
	GetParent() node
	IsLeaf() bool
	Toggle()
	GetExpand() bool
	GetUid() string
	GetLine() int
}

type treeNode struct {
	Title    string
	Parent   node
	Children []node
	Expand   bool
	Line     int
}

func (tn *treeNode) GetParent() node { return tn.Parent }
func (tn *treeNode) IsLeaf() bool    { return false }
func (tn *treeNode) Toggle()         { tn.Expand = !tn.Expand }
func (tn *treeNode) GetExpand() bool { return tn.Expand }
func (tn *treeNode) GetUid() string  { return "" }
func (tn *treeNode) GetLine() int    { return tn.Line }

type treeLeaf struct {
	Parent   node
	Resource types.Resource
	Expand   bool
	Line     int
}

func (tl *treeLeaf) GetParent() node { return tl.Parent }
func (tl *treeLeaf) IsLeaf() bool    { return true }
func (tl *treeLeaf) Toggle()         { tl.Expand = !tl.Expand }
func (tl *treeLeaf) GetExpand() bool { return tl.Expand }
func (tl *treeLeaf) GetUid() string  { return tl.Resource.GetUID() }
func (tl *treeLeaf) GetLine() int    { return tl.Line }

type TreeView struct {
	cursor     treeViewCursor
	root       *treeNode
	namespaces *treeNode
	nodes      *treeNode
	details    *treeNode
}

func NewTreeView() *TreeView {
	root := &treeNode{Title: "Root", Expand: true}
	namespaces := &treeNode{Parent: root, Expand: true, Title: "Namespaces"}
	nodes := &treeNode{Parent: root, Expand: true, Title: "Nodes"}
	details := &treeNode{Parent: root, Expand: true, Title: "Details"}
	root.Children = []node{
		namespaces,
		nodes,
		details,
	}
	return &TreeView{
		cursor:     treeViewCursor{Pos: 1},
		root:       root,
		namespaces: namespaces,
		nodes:      nodes,
		details:    details,
	}
}

func (t *TreeView) Up() {
	if t.cursor.Pos == 1 {
		return
	}
	t.cursor.Uid = ""
	t.cursor.Node = nil
	t.cursor.Pos--

	t.updateSelected()
}
func (t *TreeView) Down() {
	t.cursor.Pos++
	t.cursor.Node = nil
	t.cursor.Uid = ""

	t.updateSelected()
}
func (t *TreeView) PageUp() {
	for idx := 0; idx < 10; idx++ {
		t.Up()
	}
}
func (t *TreeView) PageDown() {
	for idx := 0; idx < 10; idx++ {
		t.Down()
	}
}
func (t *TreeView) Toggle() {
	if t.cursor.Node != nil {
		t.cursor.Node.Toggle()
	}
}

func traverseNodeTree(node node, evaluator func(node) bool) node {
	if node == nil {
		return nil
	}

	// Evaluate the current node
	if evaluator(node) {
		return node
	}

	// If the current node is a parent, traverse its children
	if !node.IsLeaf() {
		treeNode := node.(*treeNode)

		// Traverse all children of the current node
		if treeNode.Expand {
			for _, child := range treeNode.Children {
				if foundNode := traverseNodeTree(child, evaluator); foundNode != nil {
					return foundNode
				}
			}
		}
	}

	// If it's a leaf, return it if it satisfies the evaluator
	if node.IsLeaf() && evaluator(node) {
		return node
	}

	return nil
}

func (t *TreeView) findPositionOfResource(uid string) int {
	p := -1
	traverseNodeTree(t.root, func(n node) bool {
		p++
		return n.GetUid() == uid
	})
	return p
}

func (t *TreeView) findNodeAt(pos int) node {
	return traverseNodeTree(t.root, func(n node) bool {
		return n.GetLine() == pos
	})
}

func (t *TreeView) GetSelected() types.Resource {
	if val, ok := t.cursor.Node.(*treeLeaf); ok {
		return val.Resource
	}

	return nil
}

func (t *TreeView) GetSelectedLine() (int, int) {
	if t.cursor.Node == nil {
		return -1, t.cursor.Pos
	}
	return t.cursor.Node.GetLine(), t.cursor.Pos
}

func (t *TreeView) updateSelected() {
	// if len(t.cursor.Uid) != 0 {
	// 	p := t.findPositionOfResource(t.cursor.Uid)
	// 	if p != -1 {
	// 		t.cursor.Pos = p
	// 	}
	// }

	t.cursor.Node = nil
	if node := t.findNodeAt(t.cursor.Pos); node != nil {
		if node.IsLeaf() {
			t.cursor.Uid = node.(*treeLeaf).Resource.GetUID()
		}
		t.cursor.Node = node
	}
}

func (t *TreeView) Render() (string, int) {
	b := strings.Builder{}

	curLinePos := -1
	line := func(r node) string {
		curLinePos++
		if r != nil {
			if t.cursor.Pos == r.GetLine() {
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	for _, node := range []*treeNode{t.namespaces, t.nodes} {
		b.WriteString(line(node) + node.Title + "\n")
		if node.Expand {
			numOfChildren := len(node.Children)
			for idx, child := range node.Children {
				leaf := child.(*treeLeaf)
				b.WriteString(line(leaf) + " " + grommet(idx == numOfChildren-1, false) + "── " + leaf.Resource.String() + "\n")
			}
		} else {
			b.WriteString(line(nil) + "   ...\n")
		}
		b.WriteString(line(nil) + "\n")
	}

	b.WriteString(line(t.details) + t.details.Title + "\n")
	if t.details.Expand {
		for _, namespaceNode := range t.details.Children {
			namespaceTreeNode := namespaceNode.(*treeNode)

			if namespaceTreeNode.Expand {
				b.WriteString(line(namespaceTreeNode) + namespaceTreeNode.Title + "\n")
				numOfNamespaces := len(namespaceTreeNode.Children)
				for namespaceIdx, kindNode := range namespaceTreeNode.Children {
					kindTreeNode := kindNode.(*treeNode)

					if kindTreeNode.Expand {
						b.WriteString(line(kindTreeNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindTreeNode.Title + "\n")
						numOfKinds := len(kindTreeNode.Children)
						for kindIdx, resourceNode := range kindTreeNode.Children {
							resourceLeafNode := resourceNode.(*treeLeaf)
							b.WriteString(line(resourceLeafNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, true) + "   " + grommet(kindIdx == numOfKinds-1, false) + "──" + resourceLeafNode.Resource.String() + "\n")
						}
					} else {
						b.WriteString(line(kindTreeNode) + "  " + grommet(namespaceIdx == numOfNamespaces-1, false) + "── " + kindTreeNode.Title + " { ... }\n")
					}
				}
			} else {
				b.WriteString(line(namespaceTreeNode) + namespaceTreeNode.Title + "{ ... }\n")
			}

		}
	} else {
		b.WriteString(line(nil) + "   ...\n")
	}
	b.WriteString(line(nil) + "\n")

	return b.String(), t.cursor.Pos
}

// Add the resources to be rendered as a tree view
func (t *TreeView) AddResources(resourceList []types.Resource) {

	namespaces := map[string]types.Resource{}
	nodes := map[string]types.Resource{}
	other := map[string]map[string]map[string]types.Resource{}

	for _, r := range resourceList {
		switch r.GetKind() {
		case "Namespace":
			namespaces[r.GetName()] = r
		case "Node":
			nodes[r.GetName()] = r
		default:
			namespace, ok := other[r.GetNamespace()]
			if !ok {
				namespace = map[string]map[string]types.Resource{}
			}

			resourceMap, ok := namespace[r.GetKind()]
			if !ok {
				resourceMap = map[string]types.Resource{}
			}
			resourceMap[r.GetUID()] = r
			namespace[r.GetKind()] = resourceMap
			other[r.GetNamespace()] = namespace
		}
	}

	ndEnabled := map[string]bool{}
	kEnabled := map[string]map[string]bool{}
	for _, v := range t.details.Children {
		node := v.(*treeNode)
		ndEnabled[node.Title] = node.Expand
		kEnabled[node.Title] = map[string]bool{}
		for _, v2 := range node.Children {
			leaf := v2.(*treeNode)
			kEnabled[node.Title][leaf.Title] = leaf.Expand
		}
	}

	t.namespaces = &treeNode{Parent: t.root, Expand: t.namespaces.Expand, Title: "Namespaces"}
	t.nodes = &treeNode{Parent: t.root, Expand: t.nodes.Expand, Title: "Nodes"}
	t.details = &treeNode{Parent: t.root, Expand: t.details.Expand, Title: "Details"}
	t.root.Children = []node{
		t.namespaces,
		t.nodes,
		t.details,
	}
	t.root.Line = 0

	lineNo := 1
	t.namespaces.Line = lineNo
	for _, k := range slices.Sorted(maps.Keys(namespaces)) {
		lineNo++
		t.namespaces.Children = append(t.namespaces.Children, &treeLeaf{
			Parent:   t.namespaces,
			Resource: namespaces[k],
			Line:     lineNo,
		})
	}

	lineNo++
	t.nodes.Line = lineNo
	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		lineNo++
		t.nodes.Children = append(t.nodes.Children, &treeLeaf{
			Parent:   t.namespaces,
			Resource: nodes[k],
			Line:     lineNo,
		})
	}

	enabled := false
	ok := false
	lineNo++
	t.details.Line = lineNo
	for _, namespaceName := range slices.Sorted(maps.Keys(other)) {
		if enabled, ok = ndEnabled[namespaceName]; !ok {
			enabled = true
		}
		lineNo++
		namespace := &treeNode{
			Parent: t.details,
			Title:  namespaceName,
			Expand: enabled,
			Line:   lineNo,
		}
		t.details.Children = append(t.details.Children, namespace)
		for _, kindName := range slices.Sorted(maps.Keys(other[namespaceName])) {
			if enabled, ok = kEnabled[namespaceName][kindName]; !ok {
				enabled = true
			}
			lineNo++
			kind := &treeNode{
				Parent: namespace,
				Title:  kindName,
				Expand: enabled,
				Line:   lineNo,
			}
			namespace.Children = append(namespace.Children, kind)

			for _, resourceUid := range slices.Sorted(maps.Keys(other[namespaceName][kindName])) {
				lineNo++
				kind.Children = append(kind.Children, &treeLeaf{
					Parent:   kind,
					Resource: other[namespaceName][kindName][resourceUid],
					Expand:   true,
					Line:     lineNo,
				})
			}
		}

	}

}
