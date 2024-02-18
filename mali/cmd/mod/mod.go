package mod

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Mod interface {
	GetPkgFQN(filePath string) string
	GetName() string
	GetPath() string
}

func NewMod(modPath string) Mod {
	ins := &mod{}

	absPath, err := filepath.Abs(modPath)
	if err != nil {
		log.Fatal(err)
	}

	ins.path = absPath
	ins.initialize()
	return ins
}

type mod struct {
	name string
	path string
}

func (m *mod) initialize() {
	modFilePath := m.path + string(os.PathSeparator) + "go.mod"
	f, err := os.Open(modFilePath)
	if err == nil {
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			firstLine := scanner.Text()
			m.name = strings.TrimPrefix(firstLine, "module ")
		}
	} else {
		panic(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)
}

func (m *mod) GetName() string {
	return m.name
}

func (m *mod) GetPath() string {
	return m.path
}

func (m *mod) GetPkgFQN(filePath string) string {
	result := strings.Replace(filePath, m.path, m.name, 1)
	resultSlice := strings.Split(result, string(os.PathSeparator))
	if strings.Contains(resultSlice[len(resultSlice)-1], ".go") {
		resultSlice = resultSlice[:len(resultSlice)-1]
	}

	return strings.Join(resultSlice, "/")
}
