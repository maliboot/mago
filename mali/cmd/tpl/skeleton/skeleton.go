package skeleton

import _ "embed"

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

var Templates = []*Template{
	{Name: "main", Type: "go", Path: "main.go", Content: mainTxt},
	{Name: "wire", Type: "go", Path: "wire.go", Content: wireTxt},
	{Name: "conf", Type: "yml", Path: "conf.yml", Content: confTxt},
	{Name: "Makefile", Type: "", Path: "Makefile", Content: makefileTxt},
	{Name: ".gitignore", Type: "", Path: ".gitignore", Content: gitignoreTxt},
	{Name: "README", Type: "md", Path: "README.md", Content: readMeTxt},

	{Name: "config", Path: "config", IsDir: true},
	{Name: "config", Type: "go", Path: "config/config.go", Content: configTxt},
	{Name: "server", Type: "go", Path: "config/server.go", Content: serverTxt},
	{Name: "autoload", Path: "config/autoload", IsDir: true},

	{Name: "internal", Path: "internal", IsDir: true},
	{Name: "adapter", Path: "internal/adapter", IsDir: true},

	{Name: "app", Path: "internal/app", IsDir: true},
	{Name: "executor", Path: "internal/app/executor", IsDir: true},
	{Name: "command", Path: "internal/app/executor/command", IsDir: true},
	{Name: "query", Path: "internal/app/executor/query", IsDir: true},

	{Name: "client", Path: "internal/client", IsDir: true},
	{Name: "api", Path: "internal/client/api", IsDir: true},
	{Name: "dto", Path: "internal/client/dto", IsDir: true},
	{Name: "command", Path: "internal/client/dto/command", IsDir: true},
	{Name: "query", Path: "internal/client/dto/query", IsDir: true},
	{Name: "viewobject", Path: "internal/client/viewobject", IsDir: true},

	{Name: "domain", Path: "internal/domain", IsDir: true},
	{Name: "model", Path: "internal/domain/model", IsDir: true},
	{Name: "repository", Path: "internal/domain/repository", IsDir: true},
	{Name: "service", Path: "internal/domain/service", IsDir: true},

	{Name: "infra", Path: "internal/infra", IsDir: true},
	{Name: "dataobject", Path: "internal/infra/dataobject", IsDir: true},
	{Name: "repository", Path: "internal/infra/repository", IsDir: true},
}
