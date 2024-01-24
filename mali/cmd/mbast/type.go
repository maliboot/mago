package mbast

import "go/ast"

type Type int

const (
	InterfaceType Type = iota

	StructType

	FuncType

	UnknownType
)

func TypeFromString(str string) Type {
	switch str {
	case "InterfaceType":
		return InterfaceType
	case "StructType":
		return StructType
	default:
		return UnknownType
	}
}

func TypeFromExpr(expr ast.Expr) Type {
	switch expr.(type) {
	case *ast.InterfaceType:
		return InterfaceType
	case *ast.StructType:
		return StructType
	case *ast.FuncType:
		return FuncType
	default:
		return UnknownType
	}
}
