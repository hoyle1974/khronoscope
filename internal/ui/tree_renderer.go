package ui

import "strings"

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

func TreeRender(model TreeModel, cursorPos int, filter string) string {
	b := strings.Builder{}

	curLinePos := -1
	line := func(node node) string {
		curLinePos++
		if node != nil {
			if cursorPos == node.GetLine() {
				return "[*] "
			}
			return "[ ] "
		}
		return "   "
	}

	for _, node := range []*treeNode{model.namespaces, model.nodes} {
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

	b.WriteString(line(model.details) + model.details.Title + "\n")
	if model.details.Expand {
		for _, namespaceNode := range model.details.Children {
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

	return strings.Join(filterStrings(filter, strings.Split(b.String(), "\n")), "\n")

	// return b.String()
}

func filterStrings(filter string, stringsToFilter []string) []string {
	var filteredStrings []string
	for _, str := range stringsToFilter {
		if strings.Contains(str, filter) {
			filteredStrings = append(filteredStrings, str)
		}
	}
	return filteredStrings
}
