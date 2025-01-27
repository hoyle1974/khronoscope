package ui

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hoyle1974/khronoscope/internal/misc"
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
	misc.Node
	GetTitle() string
	GetParent() node
	SetParent(parent node)
	IsLeaf() bool
	Toggle()
	GetExpand() bool
	GetUid() string
	GetLine() int
	SetLine(line int)
	GetChildren() []misc.Node
}

type treeNode struct {
	Title    string
	Parent   node
	Children []node
	Expand   bool
	Line     int
	Uid      string
}

func (tn *treeNode) GetTitle() string      { return tn.Title }
func (tn *treeNode) GetParent() node       { return tn.Parent }
func (tn *treeNode) SetParent(parent node) { tn.Parent = parent }
func (tn *treeNode) IsLeaf() bool          { return false }
func (tn *treeNode) Toggle()               { tn.Expand = !tn.Expand }
func (tn *treeNode) GetExpand() bool       { return tn.Expand }
func (tn *treeNode) ShouldTraverse() bool  { return tn.Expand }
func (tn *treeNode) GetUid() string        { return tn.Uid }
func (tn *treeNode) GetLine() int          { return tn.Line }
func (tn *treeNode) SetLine(l int)         { tn.Line = l }
func (tn *treeNode) GetChildren() []misc.Node {
	b := make([]misc.Node, len(tn.Children), len(tn.Children))
	for i := range tn.Children {
		b[i] = tn.Children[i]
	}
	return b
}
func (tn *treeNode) AddChild(node node) {
	for idx, n := range tn.Children {
		if n.GetUid() == node.GetUid() {
			node.SetParent(tn)
			tn.Children[idx] = node // Replace
			return
		}
	}
	node.SetParent(tn)
	tn.Children = append(tn.Children, node)
}

type treeLeaf struct {
	Parent   node
	Resource types.Resource
	Expand   bool
	line     int
}

func (tl *treeLeaf) GetTitle() string {
	return tl.Resource.GetName() + fmt.Sprintf(":%d", tl.line)
}
func (tl *treeLeaf) GetParent() node          { return tl.Parent }
func (tl *treeLeaf) SetParent(parent node)    { tl.Parent = parent }
func (tl *treeLeaf) IsLeaf() bool             { return true }
func (tl *treeLeaf) Toggle()                  { tl.Expand = !tl.Expand }
func (tl *treeLeaf) GetExpand() bool          { return tl.Expand }
func (tl *treeLeaf) ShouldTraverse() bool     { return tl.Expand }
func (tl *treeLeaf) GetUid() string           { return tl.Resource.GetUID() }
func (tl *treeLeaf) GetLine() int             { return tl.line }
func (tl *treeLeaf) SetLine(l int)            { tl.line = l }
func (tl *treeLeaf) GetChildren() []misc.Node { return []misc.Node{} }

type TreeView struct {
	cursor     treeViewCursor
	root       *treeNode
	namespaces *treeNode
	nodes      *treeNode
	details    *treeNode
}

func NewTreeView() *TreeView {
	root := &treeNode{Title: "Root", Expand: true, Uid: uuid.New().String()}
	namespaces := &treeNode{Parent: root, Expand: true, Title: "Namespaces", Uid: uuid.New().String()}
	nodes := &treeNode{Parent: root, Expand: true, Title: "Nodes", Uid: uuid.New().String()}
	details := &treeNode{Parent: root, Expand: true, Title: "Details", Uid: uuid.New().String()}
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

func (t *TreeView) findNodeAt(pos int) node {
	return misc.TraverseNodeTree(t.root, func(n misc.Node) bool {
		return n.(node).GetLine() == pos
	}).(node)
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
	line := func(node node) string {
		curLinePos++
		if node != nil {
			if t.cursor.Pos == node.GetLine() {
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	for _, node := range []*treeNode{t.namespaces, t.nodes} {
		b.WriteString(line(node) + node.GetTitle() + "\n")
		if node.Expand {
			numOfChildren := len(node.Children)
			for idx, child := range node.Children {
				leaf := child.(*treeLeaf)
				b.WriteString(line(leaf) + " " + grommet(idx == numOfChildren-1, false) + "── " + leaf.Resource.String() + " " + leaf.GetTitle() + "\n")
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
func (t *TreeView) UpdateResources(resourceList []types.Resource) {
	// maps resource uid to the node we currently have referencing it
	nodesToDelete := map[string]node{}

	// Resources we may po
	resourcesToAdd := map[string]types.Resource{}
	for _, r := range resourceList {
		resourcesToAdd[r.GetUID()] = r
	}

	// Any resources in this list we need to update if the node already exists
	misc.TraverseNodeTree(t.root, func(n misc.Node) bool {
		if tl, ok := n.(*treeLeaf); ok {
			if res, exists := resourcesToAdd[tl.Resource.GetUID()]; exists {
				tl.Resource = res
				delete(resourcesToAdd, res.GetUID()) // We updates this resource so we don't need to add it after wards
			} else {
				nodesToDelete[tl.GetUid()] = tl // We updated this resource so the node should still exists
			}
		}
		return false
	})

	resourceList = []types.Resource{}
	for _, r := range resourcesToAdd {
		resourceList = append(resourceList, r)
	}

	// Now make a new list of resources that are
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

	for _, k := range slices.Sorted(maps.Keys(namespaces)) {
		t.namespaces.AddChild(&treeLeaf{Parent: t.namespaces, Resource: namespaces[k], Expand: true})
	}

	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		t.nodes.AddChild(&treeLeaf{Parent: t.namespaces, Resource: nodes[k], Expand: true})
	}

	for _, namespaceName := range slices.Sorted(maps.Keys(other)) {
		// Get or create a namespace node
		namespace := &treeNode{Title: namespaceName, Uid: "NS:" + namespaceName, Parent: t.details, Expand: true}
		for _, nsNodes := range t.details.Children {
			if nsNodes.GetUid() == namespace.GetUid() {
				namespace = nsNodes.(*treeNode)
				break
			}
		}
		t.details.AddChild(namespace)

		for _, kindName := range slices.Sorted(maps.Keys(other[namespaceName])) {
			// Get or create a kind node
			kind := &treeNode{Title: kindName, Uid: "NS:" + namespaceName + ".KIND:" + kindName, Parent: namespace, Expand: true}
			for _, kNodes := range namespace.Children {
				if kNodes.GetUid() == kind.GetUid() {
					kind = kNodes.(*treeNode)
					break
				}
			}
			namespace.AddChild(kind)

			for _, resourceUid := range slices.Sorted(maps.Keys(other[namespaceName][kindName])) {
				kind.AddChild(&treeLeaf{Parent: kind, Resource: other[namespaceName][kindName][resourceUid], Expand: true})
			}
		}
	}

	// remove anything that should be removed
	for _, n := range nodesToDelete {
		if n.GetParent() != nil {
			parent := n.GetParent().(*treeNode)
			for i, v := range parent.Children {
				if v.GetUid() == n.GetUid() {
					parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
					break
				}
			}
		}
	}

	// Renumber everything based on visibility
	lineNo := -1
	misc.TraverseNodeTree(t.root, func(n misc.Node) bool {
		lineNo++
		n.(node).SetLine(lineNo)
		return false
	})

}
