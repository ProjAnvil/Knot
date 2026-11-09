package services

import (
	"sort"

	models "github.com/ProjAnvil/knot/models"
)

// ParamNode is a portable representation of a parameter tree node used by
// multi-format export/import. It mirrors models.Parameter without persistence
// or ORM concerns so it can be serialized to any exchange format.
type ParamNode struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string|number|boolean|array|object
	Required    bool        `json:"required"`
	Description *string     `json:"description"`
	Order       int         `json:"order"`
	Children    []ParamNode `json:"children"`
}

// TreeFromParams converts flat ordered parameter rows (as fetched for one API
// and paramType) into a nested tree of ParamNode. Input row order is preserved
// for both roots and siblings. Rows whose ParentID references an unknown
// parameter are dropped, mirroring BuildParameterTree.
func TreeFromParams(params []models.Parameter) []ParamNode {
	if len(params) == 0 {
		return []ParamNode{}
	}

	// Index row positions by ID and group child row positions by parent,
	// both in input row order so order is preserved.
	rowByID := make(map[uint]int, len(params))
	childrenOf := make(map[uint][]int, len(params))
	var rootRows []int
	for i, p := range params {
		rowByID[p.ID] = i
		if p.ParentID == nil {
			rootRows = append(rootRows, i)
		}
	}
	for i, p := range params {
		if p.ParentID == nil {
			continue
		}
		if _, ok := rowByID[*p.ParentID]; ok {
			childrenOf[*p.ParentID] = append(childrenOf[*p.ParentID], i)
		}
	}

	// Assemble each subtree recursively from its row.
	var build func(row int) ParamNode
	build = func(row int) ParamNode {
		p := params[row]
		childRows := childrenOf[p.ID]
		children := make([]ParamNode, 0, len(childRows))
		for _, childRow := range childRows {
			children = append(children, build(childRow))
		}
		return ParamNode{
			Name:        p.Name,
			Type:        p.Type,
			Required:    p.Required,
			Description: p.Description,
			Order:       p.Order,
			Children:    children,
		}
	}

	roots := make([]ParamNode, 0, len(rootRows))
	for _, row := range rootRows {
		roots = append(roots, build(row))
	}
	return roots
}

// ParamNodesFromJSONValue converts a decoded JSON value (as produced by
// json.Unmarshal into interface{}) into ParamNodes on a best-effort basis:
// string→string, float64→number, bool→boolean, []interface{}→array,
// map[string]interface{}→object. Map keys are sorted for deterministic
// order. A non-object top-level value yields no nodes.
// Used by the Postman importer for raw JSON bodies.
func ParamNodesFromJSONValue(v interface{}) []ParamNode {
	obj, ok := v.(map[string]interface{})
	if !ok {
		return []ParamNode{}
	}

	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	nodes := make([]ParamNode, 0, len(keys))
	for i, key := range keys {
		nodes = append(nodes, paramNodeFromValue(key, i, obj[key]))
	}
	return nodes
}

// paramNodeFromValue builds a single ParamNode with the given name and order,
// inferring Type (and Children for containers) from the decoded JSON value.
func paramNodeFromValue(name string, order int, v interface{}) ParamNode {
	node := ParamNode{
		Name:     name,
		Order:    order,
		Children: []ParamNode{},
	}

	switch val := v.(type) {
	case string:
		node.Type = "string"
	case float64:
		node.Type = "number"
	case bool:
		node.Type = "boolean"
	case []interface{}:
		node.Type = "array"
		if len(val) > 0 {
			// Describe the element shape from the first element.
			if elemObj, ok := val[0].(map[string]interface{}); ok {
				node.Children = ParamNodesFromJSONValue(elemObj)
			} else {
				node.Children = []ParamNode{paramNodeFromValue("item", 0, val[0])}
			}
		}
	case map[string]interface{}:
		node.Type = "object"
		node.Children = ParamNodesFromJSONValue(val)
	default:
		// JSON null and any unexpected value degrade to string on a best-effort basis.
		node.Type = "string"
	}

	return node
}

// CountNodes returns the total number of nodes in a ParamNode tree, including
// all descendants. Used for ImportResult.ParametersImported.
func CountNodes(nodes []ParamNode) int {
	count := 0
	for _, node := range nodes {
		count += 1 + CountNodes(node.Children)
	}
	return count
}
