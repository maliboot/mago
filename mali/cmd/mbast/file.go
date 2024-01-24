package mbast

import (
	"github.com/maliboot/mago/mali/cmd/mod"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
)

type File struct {
	path string
	mod  mod.Mod
	ast  *ast.File
}

func (f *File) getPkgName() string {
	pkgSlice := strings.Split(f.path, "/")
	if len(pkgSlice) < 2 {
		return ""
	}
	return pkgSlice[len(pkgSlice)-2]
}

func (f *File) parser() ([]*Node, error) {
	inspect := inspector.New([]*ast.File{f.ast})
	pkgName := f.getPkgName()
	pkgFQN := f.mod.GetPkgFQN(f.path)

	var nodes = make([]*Node, 0)
	var parentDoc string
	inspect.Nodes([]ast.Node{&ast.GenDecl{}, &ast.TypeSpec{}, &ast.FuncDecl{}}, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			var doc Doc
			if x.Doc == nil {
				// No doc itself, so using parent's doc
				doc = Doc(parentDoc)
				parentDoc = ""
			} else {
				doc = Doc(x.Doc.Text())
			}

			nodes = append(nodes, &Node{
				Name:         x.Name.Name,
				PackageName:  pkgName,
				PackageAlias: "",
				PackageFQN:   pkgFQN,
				Type:         TypeFromExpr(x.Type),
				Attributes:   doc.ParseAttributes(),
			})
			return false
		case *ast.GenDecl:
			if !push {
				return false
			}
			if x.Tok != token.TYPE {
				return false
			}

			if !x.Lparen.IsValid() {
				if x.Doc == nil {
					return false
				}
				parentDoc = x.Doc.Text()
			}

			return true
		case *ast.FuncDecl:
			if !push {
				return false
			}
			if x.Doc == nil {
				return false
			}

			doc := Doc(x.Doc.Text())
			var attached *Node
			if x.Recv != nil {
				for _, l := range x.Recv.List {
					star, ok := l.Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					idt, ok := star.X.(*ast.Ident)
					if !ok {
						continue
					}
					for i := 0; i < len(nodes); i++ {
						if nodes[i].Type == StructType && nodes[i].Name == idt.Name {
							attached = nodes[i]
						}
					}
				}
			}
			nodes = append(nodes, &Node{
				Name:         x.Name.Name,
				PackageName:  pkgName,
				PackageAlias: "",
				PackageFQN:   pkgFQN,
				Type:         TypeFromExpr(x.Type),
				Receiver:     attached,
				Attributes:   doc.ParseAttributes(),
			})
			return false
		default:
			panic("unreachable")
		}
	})
	return nodes, nil
}
