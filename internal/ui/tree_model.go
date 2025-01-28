package ui

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/types"
)

type node interface {
	misc.Node
	GetTitle() string
	GetParent() node
	SetParent(parent node)
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
func (tl *treeLeaf) Toggle()                  { tl.Expand = !tl.Expand }
func (tl *treeLeaf) GetExpand() bool          { return tl.Expand }
func (tl *treeLeaf) ShouldTraverse() bool     { return tl.Expand }
func (tl *treeLeaf) GetUid() string           { return tl.Resource.GetUID() }
func (tl *treeLeaf) GetLine() int             { return tl.line }
func (tl *treeLeaf) SetLine(l int)            { tl.line = l }
func (tl *treeLeaf) GetChildren() []misc.Node { return []misc.Node{} }

type TreeModel struct {
	root       *treeNode
	namespaces *treeNode
	nodes      *treeNode
	details    *treeNode
}

func NewTreeModel() TreeModel {
	root := &treeNode{Title: "Root", Expand: true, Uid: uuid.New().String()}
	namespaces := &treeNode{Parent: root, Expand: true, Title: "Namespaces", Uid: uuid.New().String()}
	nodes := &treeNode{Parent: root, Expand: true, Title: "Nodes", Uid: uuid.New().String()}
	details := &treeNode{Parent: root, Expand: true, Title: "Details", Uid: uuid.New().String()}
	root.Children = []node{
		namespaces,
		nodes,
		details,
	}
	return TreeModel{
		root:       root,
		namespaces: namespaces,
		nodes:      nodes,
		details:    details,
	}
}

// Add the resources to be rendered as a tree view
func (m *TreeModel) UpdateResources(resourceList []types.Resource) {
	// maps resource uid to the node we currently have referencing it
	nodesToDelete := map[string]node{}

	// Resources we may need to add
	resourcesToAdd := map[string]types.Resource{}
	for _, r := range resourceList {
		resourcesToAdd[r.GetUID()] = r
	}

	// Any resources in this list we need to update if the node already exists
	misc.TraverseNodeTree(m.root, func(n misc.Node) bool {
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
		m.namespaces.AddChild(&treeLeaf{Parent: m.namespaces, Resource: namespaces[k], Expand: true})
	}

	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		m.nodes.AddChild(&treeLeaf{Parent: m.namespaces, Resource: nodes[k], Expand: true})
	}

	for _, namespaceName := range slices.Sorted(maps.Keys(other)) {
		// Get or create a namespace node
		namespace := &treeNode{Title: namespaceName, Uid: "NS:" + namespaceName, Parent: m.details, Expand: true}
		for _, nsNodes := range m.details.Children {
			if nsNodes.GetUid() == namespace.GetUid() {
				namespace = nsNodes.(*treeNode)
				break
			}
		}
		m.details.AddChild(namespace)

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
		m.deleteNode(n)
	}

	m.renumberNodes()

}

func (m TreeModel) deleteNode(n node) {
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

func (m TreeModel) renumberNodes() {
	lineNo := 0
	misc.TraverseNodeTree(m.root, func(n misc.Node) bool {
		n.(node).SetLine(lineNo)
		lineNo++
		return false
	})
}

func (t TreeModel) findNodeAt(pos int) node {
	return misc.TraverseNodeTree(t.root, func(n misc.Node) bool {
		return n.(node).GetLine() == pos
	}).(node)
}
