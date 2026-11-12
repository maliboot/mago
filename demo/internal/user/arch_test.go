package user_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const contextImportPath = "tests/internal/user"

// layerRules 是分层依赖规则：key 为层目录，value 为该层禁止引用的层目录。
// 约束来源：README 第 17 章「架构依赖约束」。
var layerRules = map[string][]string{
	"client":      {"application", "adapter", "domain", "infrastructure"},
	"domain":      {"application", "adapter", "infrastructure", "client"},
	"application": {"adapter", "infrastructure"},
	"adapter":     {"infrastructure"},
	// adapter/script 是组合根（与 cmd/ 同等）：迁移脚本要装配 domain/application/infrastructure，不做业务逻辑
	"adapter/script":            {},
	"infrastructure/repository": {"application", "adapter"},
	"infrastructure/convertor":  {"application", "adapter"},
	"infrastructure/util":       {"application", "adapter"},
}

// nestedLayers 记录被独立规则覆盖的子层，父层遍历时跳过，避免重复套用父层规则。
var nestedLayers = map[string][]string{
	"adapter": {"adapter/script"},
}

// TestLayerDependencyDirection 校验四层依赖方向，防止 domain、application 反向依赖外层。
func TestLayerDependencyDirection(t *testing.T) {
	for layer, forbidden := range layerRules {
		for _, file := range goFilesIn(t, filepath.FromSlash(layer), nestedLayers[layer]...) {
			for imp, line := range importsOf(t, file) {
				for _, deny := range forbidden {
					if isLayer(imp, deny) {
						t.Errorf("%s:%d 违反依赖方向：%s 层不得引用 %s 层（%s）", file, line, layer, deny, imp)
					}
				}
			}
		}
	}
}

// TestIsLayer 确认规则本身生效，避免测试因为匹配写错而永远通过。
func TestIsLayer(t *testing.T) {
	cases := []struct {
		imp   string
		layer string
		want  bool
	}{
		{"tests/internal/user/infrastructure", "infrastructure", true},
		{"tests/internal/user/application/dto", "application", true},
		{"tests/internal/user/domain", "application", false},
		{"github.com/maliboot/mago", "infrastructure", false},
		{"tests/internal/user/address/infrastructure", "infrastructure", false},
	}
	for _, c := range cases {
		if got := isLayer(c.imp, c.layer); got != c.want {
			t.Errorf("isLayer(%q, %q) = %v, want %v", c.imp, c.layer, got, c.want)
		}
	}
}

// goFilesIn 递归返回某层目录下的所有非测试 Go 文件；skip 中的子层由各自规则单独校验。
func goFilesIn(t *testing.T, dir string, skip ...string) []string {
	t.Helper()

	skipDirs := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipDirs[filepath.ToSlash(filepath.Clean(s))] = true
	}

	files := make([]string, 0, 8)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skipDirs[filepath.ToSlash(filepath.Clean(path))] {
			return filepath.SkipDir
		}
		name := d.Name()
		if d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		files = append(files, path)

		return nil
	})
	if err != nil {
		t.Fatalf("遍历目录 %s 失败：%v", dir, err)
	}

	return files
}

// importsOf 解析文件的 import 语句，返回 import 路径到行号的映射。
func importsOf(t *testing.T, file string) map[string]int {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("解析 %s 失败：%v", file, err)
	}

	result := make(map[string]int, len(f.Imports))
	for _, spec := range f.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("解析 %s 的 import 失败：%v", file, err)
		}
		result[imp] = fset.Position(spec.Pos()).Line
	}

	return result
}

// isLayer 判断 import 路径是否落在当前上下文的指定层。
func isLayer(imp, layer string) bool {
	prefix := contextImportPath + "/" + layer

	return imp == prefix || strings.HasPrefix(imp, prefix+"/")
}

// TestDomainModelFilesDeclareStruct 校验 §4.5：domain 下的模型文件（聚合根/实体/值对象）每个文件至少包含一个 struct。
// 端口文件（gateway.go 与 gateway/ 子包）是接口，不在此规则内。
func TestDomainModelFilesDeclareStruct(t *testing.T) {
	for _, file := range goFilesIn(t, filepath.FromSlash("domain")) {
		normalized := filepath.ToSlash(file)
		if strings.HasSuffix(normalized, "/gateway.go") || strings.Contains(normalized, "/gateway/") {
			continue
		}

		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s 失败：%v", file, err)
		}

		count := 0
		ast.Inspect(parsed, func(n ast.Node) bool {
			if spec, ok := n.(*ast.TypeSpec); ok {
				if _, isStruct := spec.Type.(*ast.StructType); isStruct {
					count++
				}
			}

			return true
		})

		if count == 0 {
			t.Errorf("%s 未声明 struct：聚合根/实体/值对象必须落在独立文件里（§4.5）", file)
		}
	}
}
