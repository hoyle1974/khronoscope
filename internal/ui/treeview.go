package ui

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/hoyle1974/khronoscope/resources"
)

func grommet(is bool) string {
	if !is {
		return "├"
	}
	return "└"
}

func grommet2(is bool) string {
	if !is {
		return "│"
	}
	return " "
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
	Resource resources.Resource
	Expand   bool
}

func (tl *treeLeaf) GetParent() node { return tl.Parent }
func (tl *treeLeaf) IsLeaf() bool    { return true }
func (tl *treeLeaf) Toggle()         { tl.Expand = !tl.Expand }
func (tl *treeLeaf) GetExpand() bool { return tl.Expand }
func (tl *treeLeaf) GetUid() string  { return tl.Resource.Uid }

// TreeView handles the rendering of the resource tree
type TreeView struct {
	viewport   viewport.Model
	width      int
	height     int
	cursor     treeViewCursor
	namespaces *treeNode
	nodes      *treeNode
	details    *treeNode
	resources  []resources.Resource
}

// NewTreeView creates a new tree view instance
func NewTreeView(width, height int) *TreeView {
	root := &treeNode{Title: "Root", Expand: true}
	namespaces := &treeNode{Parent: root, Expand: true, Title: "Namespaces"}
	nodes := &treeNode{Parent: root, Expand: true, Title: "Nodes"}
	details := &treeNode{Parent: root, Expand: true, Title: "Details"}
	root.Children = []node{
		namespaces,
		nodes,
		details,
	}

	tv := &TreeView{
		width:      width,
		height:     height,
		namespaces: namespaces,
		nodes:      nodes,
		details:    details,
		viewport:   viewport.New(width, height),
	}

	return tv
}

func (tv *TreeView) Update(width, height int, resources []resources.Resource, timeToUse time.Time) {
	tv.width = width
	tv.height = height
	tv.resources = resources
	tv.AddResources(resources)
	tv.viewport.Width = width
	tv.viewport.Height = height
	content, _, _ := tv.Render()
	tv.viewport.SetContent(content)
}

func (tv *TreeView) ScrollUp()   { tv.viewport.LineUp(1) }
func (tv *TreeView) ScrollDown() { tv.viewport.LineDown(1) }
func (tv *TreeView) PageUp()     { tv.viewport.HalfViewUp() }
func (tv *TreeView) PageDown()   { tv.viewport.HalfViewDown() }

func (tv *TreeView) Up() {
	if tv.cursor.Pos == 0 {
		return
	}
	tv.cursor.Uid = ""
	tv.cursor.Pos--
	content, _, _ := tv.Render()
	tv.viewport.SetContent(content)
}

func (tv *TreeView) Down() {
	tv.cursor.Pos++
	content, _, _ := tv.Render()
	tv.viewport.SetContent(content)
}

func (tv *TreeView) Toggle() {
	if tv.cursor.Node != nil {
		tv.cursor.Node.Toggle()
		content, _, _ := tv.Render()
		tv.viewport.SetContent(content)
	}
}

func (tv *TreeView) findPositionOfResource(uid string) int {
	curPos := -1
	var walk func(node node) bool
	walk = func(n node) bool {
		curPos++
		if n.IsLeaf() {
			leaf := n.(*treeLeaf)
			if leaf.Resource.Uid == uid {
				return true
			}
		} else {
			tnode := n.(*treeNode)
			if tnode.Expand {
				for _, child := range tnode.Children {
					if walk(child) {
						return true
					}
				}
			}
		}
		return false
	}

	for _, n := range []*treeNode{tv.namespaces, tv.nodes, tv.details} {
		curPos++
		if n.Expand {
			for _, child := range n.Children {
				if walk(child) {
					return curPos
				}
			}
		}
	}

	return -1
}

func (tv *TreeView) findNodeAt(pos int) node {
	curPos := -1
	var walk func(node node) node
	walk = func(n node) node {
		curPos++
		if curPos == pos {
			return n
		}
		if !n.IsLeaf() {
			tnode := n.(*treeNode)
			if tnode.Expand {
				for _, child := range tnode.Children {
					if found := walk(child); found != nil {
						return found
					}
				}
			}
		}
		return nil
	}

	for _, n := range []*treeNode{tv.namespaces, tv.nodes, tv.details} {
		curPos++
		if curPos == pos {
			return n
		}
		if n.Expand {
			for _, child := range n.Children {
				if found := walk(child); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func (tv *TreeView) Render() (string, int, *resources.Resource) {
	b := strings.Builder{}

	var retResource *resources.Resource

	curLinePos := -1
	focusLine := 0

	line := func(r node) string {
		curLinePos++
		if r != nil {
			if r == tv.cursor.Node {
				focusLine = curLinePos
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	for _, n := range []*treeNode{tv.namespaces, tv.nodes} {
		b.WriteString(line(n) + n.Title + "\n")
		if n.Expand {
			l := len(n.Children)
			for idx, child := range n.Children {
				leaf := child.(*treeLeaf)
				b.WriteString(line(leaf) + " " + grommet(idx == l-1) + "── " + leaf.Resource.String() + "\n")
			}
		} else {
			b.WriteString(line(nil) + "   ...\n")
		}
		b.WriteString(line(nil) + "\n")
	}

	b.WriteString(line(tv.details) + tv.details.Title + "\n")
	if tv.details.Expand {
		for _, namespaceNode := range tv.details.Children {
			namespaceTreeNode := namespaceNode.(*treeNode)

			if namespaceTreeNode.Expand {
				b.WriteString(line(namespaceTreeNode) + namespaceTreeNode.Title + "\n")
				l := len(namespaceTreeNode.Children)
				for idx2, kindNode := range namespaceTreeNode.Children {
					kindTreeNode := kindNode.(*treeNode)

					if kindTreeNode.Expand {
						b.WriteString(line(kindTreeNode) + "  " + grommet(idx2 == l-1) + "── " + kindTreeNode.Title + "\n")
						l2 := len(kindTreeNode.Children)
						for idx, resourceNode := range kindTreeNode.Children {
							resourceLeafNode := resourceNode.(*treeLeaf)
							b.WriteString(line(resourceLeafNode) + "  " + grommet2(idx2 == l-1) + "   " + grommet(idx == l2-1) + "──" + resourceLeafNode.Resource.String() + "\n")
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

	if focusLine < 0 {
		focusLine = 0
	} else if focusLine >= tv.viewport.Height {
		focusLine = tv.viewport.Height - 1
	}

	if node := tv.findNodeAt(tv.cursor.Pos); node != nil {
		if node.IsLeaf() {
			tv.cursor.Uid = node.(*treeLeaf).Resource.Uid
			retResource = &(node.(*treeLeaf).Resource)
		}
		tv.cursor.Node = node
	}

	return b.String(), focusLine, retResource
}

func (tv *TreeView) View() string {
	return tv.viewport.View()
}

func (tv *TreeView) ScrollPercent() float64 {
	return tv.viewport.ScrollPercent()
}

// AddResources adds resources to be rendered as a tree view
func (tv *TreeView) AddResources(resourceList []resources.Resource) {
	namespaces := map[string]resources.Resource{}
	nodes := map[string]resources.Resource{}
	other := map[string]map[string]map[string]resources.Resource{}

	for _, r := range resourceList {
		switch r.Kind {
		case "Namespace":
			namespaces[r.Name] = r
		case "Node":
			nodes[r.Name] = r
		default:
			namespace, ok := other[r.Namespace]
			if !ok {
				namespace = map[string]map[string]resources.Resource{}
			}

			resourceMap, ok := namespace[r.Kind]
			if !ok {
				resourceMap = map[string]resources.Resource{}
			}
			resourceMap[r.Uid] = r
			namespace[r.Kind] = resourceMap
			other[r.Namespace] = namespace
		}
	}

	ndEnabled := map[string]bool{}
	kEnabled := map[string]map[string]bool{}
	for _, v := range tv.details.Children {
		node := v.(*treeNode)
		ndEnabled[node.Title] = node.Expand
		kEnabled[node.Title] = map[string]bool{}
		for _, v2 := range node.Children {
			leaf := v2.(*treeNode)
			kEnabled[node.Title][leaf.Title] = leaf.Expand
		}
	}

	tv.namespaces = &treeNode{Parent: nil, Expand: tv.namespaces.Expand, Title: "Namespaces"}
	tv.nodes = &treeNode{Parent: nil, Expand: tv.nodes.Expand, Title: "Nodes"}
	tv.details = &treeNode{Parent: nil, Expand: tv.details.Expand, Title: "Details"}

	for _, k := range slices.Sorted(maps.Keys(namespaces)) {
		tv.namespaces.Children = append(tv.namespaces.Children, &treeLeaf{
			Parent:   tv.namespaces,
			Resource: namespaces[k],
		})
	}

	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		tv.nodes.Children = append(tv.nodes.Children, &treeLeaf{
			Parent:   tv.namespaces,
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
			Parent: tv.details,
			Title:  namespaceName,
			Expand: enabled,
		}
		tv.details.Children = append(tv.details.Children, namespace)
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
