package evaluator

import (
	"github.com/titivuk/go-interpreter/ast"
	"github.com/titivuk/go-interpreter/object"
)

func quote(node ast.Node) object.Object {
	return &object.Quote{Node: node}
}
