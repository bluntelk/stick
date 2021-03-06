package stick

import (
	"strings"

	"github.com/tyler-sommer/stick/escape"
	"github.com/tyler-sommer/stick/parse"
)

// AutoEscapeExtension provides Twig equivalent escaping for Stick templates.
type AutoEscapeExtension struct {
	Escapers map[string]escape.Escaper
}

// Init registers the escape functionality with the given Env.
func (e *AutoEscapeExtension) Init(env *Env) error {
	env.Visitors = append(env.Visitors, &autoEscapeVisitor{})
	env.Filters["escape"] = func(ctx Context, val Value, args ...Value) Value {
		ct := "html"
		if len(args) > 0 {
			ct = CoerceString(args[0])
		}

		if sval, ok := val.(SafeValue); ok {
			if sval.IsSafe(ct) {
				return val
			}
		}

		escfn, ok := e.Escapers[ct]
		if !ok {
			// TODO: Communicate error
			return NewSafeValue("", ct)
		}

		return NewSafeValue(escfn(CoerceString(val)), ct)
	}
	return nil
}

// NewAutoEscapeExtension returns an AutoEscapeExtension with Twig equivalent
// Escapers, by default.
func NewAutoEscapeExtension() *AutoEscapeExtension {
	return &AutoEscapeExtension{
		Escapers: map[string]escape.Escaper{
			"html":      escape.HTML,
			"html_attr": escape.HTMLAttribute,
			"js":        escape.JS,
			"css":       escape.CSS,
			"url":       escape.URLQueryParam,
		},
	}
}

// AutoEscapeVisitor can be used to automatically apply the "escape" filter
// to any PrintNode.
type autoEscapeVisitor struct {
	stack []string
}

// push adds the given name on top of the stack.
func (v *autoEscapeVisitor) push(name string) {
	v.stack = append(v.stack, name)
}

// pop removes the top-most name on the stack.
func (v *autoEscapeVisitor) pop() {
	v.stack = v.stack[0 : len(v.stack)-1]
}

func (v *autoEscapeVisitor) current() string {
	if len(v.stack) == 0 {
		// TODO: This is an invalid state.
		return ""
	}
	return v.stack[len(v.stack)-1]
}

func (v *autoEscapeVisitor) Enter(n parse.Node) {
	switch node := n.(type) {
	case *parse.ModuleNode:
		v.push(v.guessTypeFromName(node.Origin))
	case *parse.BlockNode:
		v.push(v.guessTypeFromName(node.Origin))
	case *parse.PrintNode:
		ct := v.current()
		v := node.X
		r := parse.NewFilterExpr(
			"escape",
			[]parse.Expr{v, parse.NewStringExpr(ct, v.Start())},
			v.Start(),
		)
		node.X = r
	}
}

func (v *autoEscapeVisitor) Leave(n parse.Node) {
	switch n.(type) {
	case *parse.ModuleNode, *parse.BlockNode:
		v.pop()
	}
}

func (v *autoEscapeVisitor) guessTypeFromName(name string) string {
	name = strings.TrimSuffix(name, ".twig")
	p := strings.LastIndex(name, ".")
	if p < 0 {
		// Default to html
		return "html"
	}
	return name[p:]
}
