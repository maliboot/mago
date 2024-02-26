package helper

import (
	"fmt"
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
	var s = new(S)
	err := sonic.Unmarshal(json, s)
	return s, err
}

func ToLowerCamelJson(snakeJson []byte) ([]byte, error) {
	root, err := sonic.Get(snakeJson)
	if err != nil {
		return nil, err
	}

	if !root.Valid() {
		return nil, fmt.Errorf("snakeJson转camelJson解析失败，json:%s", snakeJson)
	}

	newRoot := recursionJsonNode(root, func(key string) string {
		if !strings.Contains(key, "_") {
			return key
		}
		return strcase.ToLowerCamel(key)
	})
	return newRoot.MarshalJSON()
}

func recursionJsonNode(root ast.Node, keyFunc func(key string) string) ast.Node {
	_ = root.ForEach(func(path ast.Sequence, node *ast.Node) bool {
		if path.Index < 0 {
			return false
		}

		if path.Key != nil && keyFunc != nil {
			root.IndexPair(path.Index).Key = keyFunc(*path.Key)
		}

		nodeType := node.Type()
		if nodeType == 5 || nodeType == 6 {
			_, _ = root.SetByIndex(path.Index, recursionJsonNode(*node, keyFunc))
		}
		return true
	})

	return root
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
