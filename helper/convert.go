package helper

import (
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/iancoleman/strcase"
)

func Marshal(object interface{}) ([]byte, error) {
	var output []byte
	output, err := sonic.Marshal(object)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func Unmarshal[S any](json []byte) (*S, error) {
	root, err := sonic.Get(json)
	if err != nil {
		return nil, err
	}

	_ = root.ForEach(func(path ast.Sequence, node *ast.Node) bool {
		// snake to camel
		if strings.Contains(*path.Key, "_") {
			root.IndexPair(path.Index).Key = strcase.ToLowerCamel(*path.Key)
		}
		return true
	})

	var s = new(S)
	err = sonic.Unmarshal(json, s)
	return s, err
}

func ToLowerCamelJson(snakeJson []byte) ([]byte, error) {
	root, err := sonic.Get(snakeJson)
	if err != nil {
		return nil, err
	}
	_ = root.ForEach(func(path ast.Sequence, node *ast.Node) bool {
		if path.Index >= 0 && strings.Contains(*path.Key, "_") {
			root.IndexPair(path.Index).Key = strcase.ToLowerCamel(*path.Key)
		}
		return true
	})

	return root.MarshalJSON()
}

func Convertor[S any](source interface{}) (*S, error) {
	if source == nil {
		return nil, nil
	}
	json, err := Marshal(source)
	if err != nil {
		return nil, err
	}

	camelJson, err := ToLowerCamelJson(json)
	if err != nil {
		return nil, err
	}

	des, err := Unmarshal[S](camelJson)
	if err != nil {
		return nil, err
	}

	return des, err
}
