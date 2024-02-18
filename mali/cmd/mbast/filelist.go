package mbast

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/maliboot/mago/mali/cmd/mod"
)

type Files []*File

func NewFiles(modIns mod.Mod) (Files, error) {
	var result = make([]*File, 0)
	err := filepath.Walk(modIns.GetPath(), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		fSet := token.NewFileSet()
		fAst, err := parser.ParseFile(fSet, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		result = append(result, &File{
			path: path,
			mod:  modIns,
			ast:  fAst,
		})
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

func (fs Files) Parser() (Nodes, error) {
	nodes := make(Nodes, 0)
	for i := 0; i < len(fs); i++ {
		parserNodes, err := fs[i].parser()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, parserNodes...)
	}
	return nodes, nil
}
