package ui

import (
	"github.com/hoyle1974/khronoscope/internal/types"
)

// TreeView provides a way to browse a set of k8s resources in a tree view.
// It builds a view consisting of 3 sections: namespaces, nodes, and details.
// It manages cursor movement in the view, collapsing/expanding nodes and tries
// to keep the cursor mostly sane even when resources the cursor is on disappear.

type treeViewCursor struct {
	Pos  int
	Uid  string
	Node node
}

type TreeController struct {
	cursor treeViewCursor
	model  TreeModel
	filter string
}

func NewTreeView() *TreeController {
	return &TreeController{
		cursor: treeViewCursor{Pos: 1},
		model:  NewTreeModel(),
	}
}

func (t *TreeController) SetFilter(filter string) {
	t.filter = filter
}

func (t *TreeController) Up() {
	if t.cursor.Pos == 1 {
		return
	}
	t.cursor.Uid = ""
	t.cursor.Node = nil
	t.cursor.Pos--

	t.updateSelected()
}
func (t *TreeController) Down() {
	t.cursor.Pos++
	t.cursor.Node = nil
	t.cursor.Uid = ""

	t.updateSelected()
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
		t.cursor.Node.Toggle()
	}
}

func (t *TreeController) GetSelected() types.Resource {
	if val, ok := t.cursor.Node.(*treeLeaf); ok {
		return val.Resource
	}

	return nil
}

func (t *TreeController) GetSelectedLine() (int, int) {
	if t.cursor.Node == nil {
		return -1, t.cursor.Pos
	}
	return t.cursor.Node.GetLine(), t.cursor.Pos
}

func (t *TreeController) updateSelected() {
	t.cursor.Node = nil
	if node := t.model.findNodeAt(t.cursor.Pos); node != nil {
		t.cursor.Uid = node.GetUid()
		t.cursor.Node = node
	}
}

func (t *TreeController) Render() (string, int) {
	return TreeRender(t.model, t.cursor.Pos, t.filter), t.cursor.Pos
}

func (t *TreeController) UpdateResources(resources []types.Resource) {
	t.model.UpdateResources(resources)
}
