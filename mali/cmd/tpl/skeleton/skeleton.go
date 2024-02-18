package skeleton

import (
	_ "embed"
	"os"
)

var (
	//go:embed main.go.tmpl
	mainTxt string

	//go:embed conf.yml.tmpl
	confTxt string

	//go:embed Makefile.tmpl
	makefileTxt string

	//go:embed .gitignore.tmpl
	gitignoreTxt string

	//go:embed README.md.tmpl
	readMeTxt string

	//go:embed wire.go.tmpl
	wireTxt string

	//go:embed config/config.go.tmpl
	configTxt string

	//go:embed config/server.go.tmpl
	serverTxt string
)

type Template struct {
	Name    string
	Type    string
	Path    string
	IsDir   bool
	Content string
}

var ps = string(os.PathSeparator)

var Templates = []*Template{
	{Name: "main", Type: "go", Path: "main.go", Content: mainTxt},
	{Name: "wire", Type: "go", Path: "wire.go", Content: wireTxt},
	{Name: "conf", Type: "yml", Path: "conf.yml", Content: confTxt},
	{Name: "Makefile", Type: "", Path: "Makefile", Content: makefileTxt},
	{Name: ".gitignore", Type: "", Path: ".gitignore", Content: gitignoreTxt},
	{Name: "README", Type: "md", Path: "README.md", Content: readMeTxt},

	{Name: "config", Path: "config", IsDir: true},
	{Name: "config", Type: "go", Path: "config" + ps + "config.go", Content: configTxt},
	{Name: "server", Type: "go", Path: "config" + ps + "server.go", Content: serverTxt},
	{Name: "autoload", Path: "config" + ps + "autoload", IsDir: true},

	{Name: "internal", Path: "internal", IsDir: true},
	{Name: "adapter", Path: "internal" + ps + "adapter", IsDir: true},

	{Name: "app", Path: "internal" + ps + "app", IsDir: true},
	{Name: "executor", Path: "internal" + ps + "app" + ps + "executor", IsDir: true},
	{Name: "command", Path: "internal" + ps + "app" + ps + "executor" + ps + "command", IsDir: true},
	{Name: "query", Path: "internal" + ps + "app" + ps + "executor" + ps + "query", IsDir: true},

	{Name: "client", Path: "internal" + ps + "client", IsDir: true},
	{Name: "api", Path: "internal" + ps + "client" + ps + "api", IsDir: true},
	{Name: "dto", Path: "internal" + ps + "client" + ps + "dto", IsDir: true},
	{Name: "command", Path: "internal" + ps + "client" + ps + "dto" + ps + "command", IsDir: true},
	{Name: "query", Path: "internal" + ps + "client" + ps + "dto" + ps + "query", IsDir: true},
	{Name: "viewobject", Path: "internal" + ps + "client" + ps + "viewobject", IsDir: true},

	{Name: "domain", Path: "internal" + ps + "domain", IsDir: true},
	{Name: "model", Path: "internal" + ps + "domain" + ps + "model", IsDir: true},
	{Name: "repository", Path: "internal" + ps + "domain" + ps + "repository", IsDir: true},
	{Name: "service", Path: "internal" + ps + "domain" + ps + "service", IsDir: true},

	{Name: "infra", Path: "internal" + ps + "infra", IsDir: true},
	{Name: "dataobject", Path: "internal" + ps + "infra" + ps + "dataobject", IsDir: true},
	{Name: "repository", Path: "internal" + ps + "infra" + ps + "repository", IsDir: true},
}
