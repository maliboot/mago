package mago

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// allowedSuffix 是 §16.7 允许的下划线后缀：测试文件、GOOS/GOARCH 构建约束。
var allowedSuffix = regexp.MustCompile(`(?i)_(test|windows|linux|darwin|freebsd|openbsd|netbsd|android|ios|js|wasip1|plan9|illumos|dragonfly|amd64|arm64|arm|386|ppc64|ppc64le|mips|mipsle|mips64|mips64le|s390x|riscv64|loong64|wasm)$`)

// generatedStems 是代码生成器固定输出名：工具不接受改名，是本规范的唯一例外（§16.7）。
// wire_gen.go 由 `go tool wire` 生成，禁止手改。
var generatedStems = map[string]bool{
	"wire_gen": true,
}

// namingViolation 判定文件名主干是否违反 §16.7。
func namingViolation(stem string) bool {
	if generatedStems[stem] {
		return false
	}

	base := allowedSuffix.ReplaceAllString(stem, "")

	return strings.Contains(base, "_") || base != strings.ToLower(base)
}

// skipDirs 是不参与命名扫描的目录。
var skipDirs = map[string]bool{
	".git": true, ".idea": true, "runtime": true, "node_modules": true, "vendor": true,
}

// TestGoFileNaming 守护 §16.7 文件命名规范：
// Go 文件名默认全小写、单词直连，下划线只允许出现在 _test / _windows / _linux 这类特殊后缀里。
func TestGoFileNaming(t *testing.T) {
	var checked int

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}

			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		stem := strings.TrimSuffix(name, ".go")
		// 以 _ 或 . 开头的文件被 go 工具链忽略，不受本规范约束
		if strings.HasPrefix(stem, "_") || strings.HasPrefix(stem, ".") {
			return nil
		}

		checked++

		if namingViolation(stem) {
			t.Errorf("%s: 文件名不符合 §16.7，应改为 %s.go（下划线只允许 _test/_windows/_linux 等特殊后缀，生成文件只有 wire_gen.go）", path, strings.ReplaceAll(stem, "_", ""))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("遍历目录失败：%v", err)
	}

	if checked == 0 {
		t.Fatal("没有扫描到任何 Go 文件，命名守护未生效")
	}
}

// TestGoFileNamingRule 校验规则本身能识别违规写法，避免守护因为匹配写错而永远通过。
func TestGoFileNamingRule(t *testing.T) {
	cases := []struct {
		stem string
		bad  bool
	}{
		{"user", false},
		{"useraddress", false},
		{"user_test", false},
		{"livesmtp_test", false},
		{"tool_windows", false},
		{"wire_gen", false},
		{"wire", false},
		{"user_address", true},
		{"live_smtp_test", true},
		{"my_gen", true},
		{"User", true},
	}

	for _, c := range cases {
		if got := namingViolation(c.stem); got != c.bad {
			t.Errorf("规则判定 %q = %v, 期望 %v", c.stem, got, c.bad)
		}
	}
}
