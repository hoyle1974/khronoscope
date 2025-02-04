package ui

import (
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/types"
)

// TreeView provides a way to browse a set of k8s resources in a tree view.
// It builds a view consisting of 3 sections: namespaces, nodes, and details.
// It manages cursor movement in the view, collapsing/expanding nodes and tries
// to keep the cursor mostly sane even when resources the cursor is on disappear.

type treeViewCursor struct {
	Pos  int
	Uid  string
	Node *renderNode
}

type TreeController struct {
	cursor treeViewCursor
	model  TreeModel
	filter Filter
}

func NewTreeView() *TreeController {
	return &TreeController{
		cursor: treeViewCursor{Pos: 1},
		model:  NewTreeModel(),
	}
}

func (t *TreeController) SetFilter(filter Filter) {
	t.filter = filter
}

func (t *TreeController) Up() {
	if t.cursor.Pos == 1 {
		return
	}
	t.cursor.Uid = ""
	t.cursor.Node = nil
	t.cursor.Pos--
}
func (t *TreeController) Down() {
	t.cursor.Pos++
	t.cursor.Node = nil
	t.cursor.Uid = ""
}
func (t *TreeController) PageUp() {
	for idx := 0; idx < 10; idx++ {
		t.Up()
	}
}
func (t *TreeController) PageDown() {
	for idx := 0; idx < 10; idx++ {
		t.Down()
	}
}
func (t *TreeController) Toggle() {
	if t.cursor.Node != nil {
		t.cursor.Node.Node.Toggle()
	}
}

func (t *TreeController) GetSelected() types.Resource {
	if t.cursor.Node == nil || t.cursor.Node.Node == nil {
		return nil
	}
	if val, ok := t.cursor.Node.Node.(*treeLeaf); ok {
		return val.Resource
	}

	return nil
}

func (t *TreeController) GetSelectedLine() (int, int) {
	if t.cursor.Node == nil {
		return -1, t.cursor.Pos
	}
	return t.cursor.Node.Line, t.cursor.Pos
}

func (t *TreeController) Render(vcrEnabled bool) (string, int) {
	root, max := createRenderTree(t.model, t.filter)

	ret := misc.TraverseNodeTree(root, func(n misc.Node) bool {
		return n.(*renderNode).Line == t.cursor.Pos
	})
	if ret != nil {
		t.cursor.Node = ret.(*renderNode)
		t.cursor.Uid = t.cursor.Node.Node.GetUid()
	}
	if t.cursor.Pos > max-1 {
		t.cursor.Pos = max - 1
	}

	return treeRender(root, vcrEnabled, t.cursor.Pos, t.filter), t.cursor.Pos
}

func (t *TreeController) UpdateResources(resources []types.Resource) {
	t.model.UpdateResources(resources)
}
