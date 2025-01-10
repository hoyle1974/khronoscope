package main

import (
	"maps"
	"slices"
	"strings"
)

type TreeViewCursor struct {
	Pos  int
	Uid  string
	Node Node
}

type Node interface {
	GetParent() Node
	IsLeaf() bool
	Toggle()
	GetExpand() bool
	GetUid() string
}

type TreeNode struct {
	Title    string
	Parent   Node
	Children []Node
	Expand   bool
}

func (tn *TreeNode) GetParent() Node { return tn.Parent }
func (tn *TreeNode) IsLeaf() bool    { return false }
func (tn *TreeNode) Toggle()         { tn.Expand = !tn.Expand }
func (tn *TreeNode) GetExpand() bool { return tn.Expand }
func (tn *TreeNode) GetUid() string  { return "" }

type TreeLeaf struct {
	Parent   Node
	Resource Resource
	Expand   bool
}

func (tl *TreeLeaf) GetParent() Node { return tl.Parent }
func (tl *TreeLeaf) IsLeaf() bool    { return true }
func (tl *TreeLeaf) Toggle()         { tl.Expand = !tl.Expand }
func (tl *TreeLeaf) GetExpand() bool { return tl.Expand }
func (tl *TreeLeaf) GetUid() string  { return tl.Resource.Uid }

type TreeView struct {
	cursor     TreeViewCursor
	namespaces *TreeNode
	nodes      *TreeNode
	details    *TreeNode
	root       *TreeNode
}

func NewTreeView() *TreeView {
	root := &TreeNode{Title: "Root"}
	namespaces := &TreeNode{Parent: root, Expand: false, Title: "Namespaces"}
	nodes := &TreeNode{Parent: root, Expand: false, Title: "Nodes"}
	details := &TreeNode{Parent: root, Expand: true, Title: "Details"}
	root.Children = []Node{
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
				for _, n2 := range n1.(*TreeNode).Children {
					idx++
					if n2.GetExpand() {
						for _, n3 := range n2.(*TreeNode).Children {
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

func (t *TreeView) findNodeAt(pos int) Node {
	idx := -1
	var retN Node

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
				for _, n2 := range n1.(*TreeNode).Children {
					idx++
					if idx == pos {
						retN = n2
					}
					if n2.GetExpand() {
						for _, n3 := range n2.(*TreeNode).Children {
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

func (t *TreeView) Render() (string, int, *Resource) {
	b := strings.Builder{}

	var retResource *Resource

	if len(t.cursor.Uid) != 0 {
		p := t.findPositionOfResource(t.cursor.Uid)
		if p != -1 {
			t.cursor.Pos = p
		}
	}

	if node := t.findNodeAt(t.cursor.Pos); node != nil {
		if node.IsLeaf() {
			t.cursor.Uid = node.(*TreeLeaf).Resource.Uid
			retResource = &(node.(*TreeLeaf).Resource)
		}
		t.cursor.Node = node
	}

	// b.WriteString(fmt.Sprintf("Cursor: %d [%s] Expand:%v %v\n", t.cursor.Pos, t.cursor.Uid, t.cursor.Node.GetExpand(), t.cursor.Node.IsLeaf()))

	curLinePos := -1
	focusLine := 0

	line := func(r Node) string {
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

	for _, n := range []*TreeNode{t.namespaces, t.nodes} {

		b.WriteString(line(n) + n.Title + "\n")
		if n.Expand {
			l := len(n.Children)
			for idx, child := range n.Children {
				leaf := child.(*TreeLeaf)

				render := leaf.Resource.String()
				if len(render) == 0 {
					b.WriteString(line(leaf) + " " + grommet(idx == l-1) + "── " + leaf.Resource.Name + "\n")
				} else {
					for idx2, s := range render {
						if idx2 == 0 {
							b.WriteString(line(leaf) + " " + grommet(idx == l-1) + "── " + s + "\n")
						} else {
							b.WriteString(line(leaf) + " │  " + s + "\n")
						}
					}
				}
			}
		} else {
			b.WriteString(line(nil) + "   ...\n")
		}
		b.WriteString(line(nil) + "\n")
	}

	b.WriteString(line(t.details) + t.details.Title + "\n")
	if t.details.Expand {
		for _, namespaceNode := range t.details.Children {
			namespaceTreeNode := namespaceNode.(*TreeNode)

			if namespaceTreeNode.Expand {
				b.WriteString(line(namespaceTreeNode) + namespaceTreeNode.Title + "\n")
				l := len(namespaceTreeNode.Children)
				for idx2, kindNode := range namespaceTreeNode.Children {
					kindTreeNode := kindNode.(*TreeNode)

					if kindTreeNode.Expand {
						b.WriteString(line(kindTreeNode) + "  " + grommet(idx2 == l-1) + "── " + kindTreeNode.Title + "\n")
						l2 := len(kindTreeNode.Children)
						for idx, resourceNode := range kindTreeNode.Children {
							resourceLeafNode := resourceNode.(*TreeLeaf)
							render := resourceLeafNode.Resource.String()
							txt := ""
							if len(render) == 0 {
								txt = resourceLeafNode.Resource.Name
							} else {
								txt = strings.Join(render, " ")
							}
							b.WriteString(line(resourceLeafNode) + "  " + grommet2(idx2 == l-1) + "   " + grommet(idx == l2-1) + "──" + txt + "\n")
						}
					} else {
						b.WriteString(line(kindTreeNode) + "  " + grommet(idx2 == l-1) + "── " + kindTreeNode.Title + " { ... }\n")
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
func (t *TreeView) AddResources(resources []Resource) {
	namespaces := map[string]Resource{}
	nodes := map[string]Resource{}
	other := map[string]map[string]map[string]Resource{}

	for _, r := range resources {
		switch r.Kind {
		case "Namespace":
			namespaces[r.Name] = r
		case "Node":
			nodes[r.Name] = r
		default:
			namespace, ok := other[r.Namespace]
			if !ok {
				namespace = map[string]map[string]Resource{}
			}

			resourceMap, ok := namespace[r.Kind]
			if !ok {
				resourceMap = map[string]Resource{}
			}
			resourceMap[r.Uid] = r
			namespace[r.Kind] = resourceMap
			other[r.Namespace] = namespace
		}
	}

	ndEnabled := map[string]bool{}
	kEnabled := map[string]map[string]bool{}
	for _, v := range t.details.Children {
		node := v.(*TreeNode)
		ndEnabled[node.Title] = node.Expand
		kEnabled[node.Title] = map[string]bool{}
		for _, v2 := range node.Children {
			leaf := v2.(*TreeNode)
			kEnabled[node.Title][leaf.Title] = leaf.Expand
		}
	}

	t.namespaces = &TreeNode{Parent: t.root, Expand: t.namespaces.Expand, Title: "Namespaces"}
	t.nodes = &TreeNode{Parent: t.root, Expand: t.nodes.Expand, Title: "Nodes"}
	t.details = &TreeNode{Parent: t.root, Expand: t.details.Expand, Title: "Details"}

	for _, k := range slices.Sorted(maps.Keys(namespaces)) {
		t.namespaces.Children = append(t.namespaces.Children, &TreeLeaf{
			Parent:   t.namespaces,
			Resource: namespaces[k],
		})
	}

	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		t.nodes.Children = append(t.nodes.Children, &TreeLeaf{
			Parent:   t.namespaces,
			Resource: nodes[k],
		})
	}

	enabled := false
	ok := false
	for _, namespaceName := range slices.Sorted(maps.Keys(other)) {
		if enabled, ok = ndEnabled[namespaceName]; !ok {
			enabled = false
		}
		namespace := &TreeNode{
			Parent: t.details,
			Title:  namespaceName,
			Expand: enabled,
		}
		t.details.Children = append(t.details.Children, namespace)
		for _, kindName := range slices.Sorted(maps.Keys(other[namespaceName])) {
			if enabled, ok = kEnabled[namespaceName][kindName]; !ok {
				enabled = true
			}
			kind := &TreeNode{
				Parent: namespace,
				Title:  kindName,
				Expand: enabled,
			}
			namespace.Children = append(namespace.Children, kind)

			for _, resourceUid := range slices.Sorted(maps.Keys(other[namespaceName][kindName])) {
				kind.Children = append(kind.Children, &TreeLeaf{
					Parent:   kind,
					Resource: other[namespaceName][kindName][resourceUid],
					Expand:   true,
				})
			}
		}

	}

}
