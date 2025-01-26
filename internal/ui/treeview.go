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
}

type treeNode struct {
	Title    string
	Parent   node
	Children []node
	Expand   bool
}

func (tn *treeNode) GetParent() node { return tn.Parent }
func (tn *treeNode) IsLeaf() bool    { return false }
func (tn *treeNode) Toggle()         { tn.Expand = !tn.Expand }
func (tn *treeNode) GetExpand() bool { return tn.Expand }
func (tn *treeNode) GetUid() string  { return "" }

type treeLeaf struct {
	Parent   node
	Resource types.Resource
	Expand   bool
}

func (tl *treeLeaf) GetParent() node { return tl.Parent }
func (tl *treeLeaf) IsLeaf() bool    { return true }
func (tl *treeLeaf) Toggle()         { tl.Expand = !tl.Expand }
func (tl *treeLeaf) GetExpand() bool { return tl.Expand }
func (tl *treeLeaf) GetUid() string  { return tl.Resource.GetUID() }

type TreeView struct {
	cursor     treeViewCursor
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
		namespaces: namespaces,
		nodes:      nodes,
		details:    details,
	}
}

func (t *TreeView) Up() {
	if t.cursor.Pos == 0 {
		return
	}
	t.cursor.Uid = ""
	t.cursor.Pos--

}
func (t *TreeView) Down() {
	t.cursor.Pos++
	t.cursor.Uid = ""
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

func (t *TreeView) findPositionOfResource(uid string) int {
	idx := -1

	idx++

	if t.namespaces.Expand {
		for _, n := range t.namespaces.Children {
			idx++
			if uid == n.GetUid() {
				return idx
			}
		}
	}

	idx++

	if t.nodes.Expand {

		for _, n := range t.nodes.Children {
			idx++
			if uid == n.GetUid() {
				return idx
			}

		}
	}

	idx++

	if t.details.Expand {
		for _, n1 := range t.details.Children {
			idx++
			if n1.GetExpand() {
				for _, n2 := range n1.(*treeNode).Children {
					idx++
					if n2.GetExpand() {
						for _, n3 := range n2.(*treeNode).Children {
							idx++
							if uid == n3.GetUid() {
								return idx
							}
						}
					}
				}
			}
		}
	}

	return -1
}

func (t *TreeView) findNodeAt(pos int) node {
	idx := -1
	var retN node

	idx++
	if idx == pos {
		return t.namespaces
	}

	if t.namespaces.Expand {
		for _, n := range t.namespaces.Children {
			idx++
			if idx == pos {
				retN = n
			}
		}
	}

	idx++
	if idx == pos {
		retN = t.nodes
	}

	if t.nodes.Expand {

		for _, n := range t.nodes.Children {
			idx++
			if idx == pos {
				retN = n
			}
		}
	}

	idx++
	if idx == pos {
		retN = t.details
	}

	if t.details.Expand {
		for _, n1 := range t.details.Children {
			idx++
			if idx == pos {
				retN = n1
			}
			if n1.GetExpand() {
				for _, n2 := range n1.(*treeNode).Children {
					idx++
					if idx == pos {
						retN = n2
					}
					if n2.GetExpand() {
						for _, n3 := range n2.(*treeNode).Children {
							idx++
							if idx == pos {
								retN = n3
							}
						}
					}
				}
			}
		}
	}

	if t.cursor.Pos > idx {
		t.cursor.Pos = idx
	}

	return retN
}

func (t *TreeView) Render() (string, int, types.Resource) {
	b := strings.Builder{}

	var retResource types.Resource

	if len(t.cursor.Uid) != 0 {
		p := t.findPositionOfResource(t.cursor.Uid)
		if p != -1 {
			t.cursor.Pos = p
		}
	}

	if node := t.findNodeAt(t.cursor.Pos); node != nil {
		if node.IsLeaf() {
			t.cursor.Uid = node.(*treeLeaf).Resource.GetUID()
			retResource = (node.(*treeLeaf).Resource)
		}
		t.cursor.Node = node
	}

	// b.WriteString(fmt.Sprintf("Cursor: %d [%s] Expand:%v %v\n", t.cursor.Pos, t.cursor.Uid, t.cursor.Node.GetExpand(), t.cursor.Node.IsLeaf()))

	curLinePos := -1
	focusLine := 0

	line := func(r node) string {
		curLinePos++
		if r != nil {
			if r == t.cursor.Node {
				focusLine = curLinePos
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	for _, n := range []*treeNode{t.namespaces, t.nodes} {
		b.WriteString(line(n) + n.Title + "\n")
		if n.Expand {
			l := len(n.Children)
			for idx, child := range n.Children {
				leaf := child.(*treeLeaf)
				b.WriteString(line(leaf) + " " + grommet(idx == l-1, false) + "── " + leaf.Resource.String() + "\n")
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
				l := len(namespaceTreeNode.Children)
				for idx2, kindNode := range namespaceTreeNode.Children {
					kindTreeNode := kindNode.(*treeNode)

					if kindTreeNode.Expand {
						b.WriteString(line(kindTreeNode) + "  " + grommet(idx2 == l-1, false) + "── " + kindTreeNode.Title + "\n")
						l2 := len(kindTreeNode.Children)
						for idx, resourceNode := range kindTreeNode.Children {
							resourceLeafNode := resourceNode.(*treeLeaf)
							b.WriteString(line(resourceLeafNode) + "  " + grommet(idx2 == l-1, true) + "   " + grommet(idx == l2-1, false) + "──" + resourceLeafNode.Resource.String() + "\n")
						}
					} else {
						b.WriteString(line(kindTreeNode) + "  " + grommet(idx2 == l-1, false) + "── " + kindTreeNode.Title + " { ... }\n")
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

	return b.String(), focusLine, retResource
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

	t.namespaces = &treeNode{Parent: nil, Expand: t.namespaces.Expand, Title: "Namespaces"}
	t.nodes = &treeNode{Parent: nil, Expand: t.nodes.Expand, Title: "Nodes"}
	t.details = &treeNode{Parent: nil, Expand: t.details.Expand, Title: "Details"}

	for _, k := range slices.Sorted(maps.Keys(namespaces)) {
		t.namespaces.Children = append(t.namespaces.Children, &treeLeaf{
			Parent:   t.namespaces,
			Resource: namespaces[k],
		})
	}

	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		t.nodes.Children = append(t.nodes.Children, &treeLeaf{
			Parent:   t.namespaces,
			Resource: nodes[k],
		})
	}

	enabled := false
	ok := false
	for _, namespaceName := range slices.Sorted(maps.Keys(other)) {
		if enabled, ok = ndEnabled[namespaceName]; !ok {
			enabled = true
		}
		namespace := &treeNode{
			Parent: t.details,
			Title:  namespaceName,
			Expand: enabled,
		}
		t.details.Children = append(t.details.Children, namespace)
		for _, kindName := range slices.Sorted(maps.Keys(other[namespaceName])) {
			if enabled, ok = kEnabled[namespaceName][kindName]; !ok {
				enabled = true
			}
			kind := &treeNode{
				Parent: namespace,
				Title:  kindName,
				Expand: enabled,
			}
			namespace.Children = append(namespace.Children, kind)

			for _, resourceUid := range slices.Sorted(maps.Keys(other[namespaceName][kindName])) {
				kind.Children = append(kind.Children, &treeLeaf{
					Parent:   kind,
					Resource: other[namespaceName][kindName][resourceUid],
					Expand:   true,
				})
			}
		}

	}

}
