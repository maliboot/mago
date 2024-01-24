package skeleton

import _ "embed"

var (
	//go:embed internal/adapter/admin/controller.go.tmpl
	controllerTxt string

	//go:embed internal/app/executor/command/createcmdexe.go.tmpl
	createCmdExeTxt string

	//go:embed internal/app/executor/command/updatecmdexe.go.tmpl
	updateCmdExeTxt string

	//go:embed internal/app/executor/command/deletecmdexe.go.tmpl
	deleteCmdExeTxt string

	//go:embed internal/app/executor/query/getbyidqryexe.go.tmpl
	getByIdQryExeTxt string

	//go:embed internal/app/executor/query/listbypageqryexe.go.tmpl
	listByPageQryExeTxt string

	//go:embed internal/client/dto/command/createcmd.go.tmpl
	createCmdTxt string

	//go:embed internal/client/dto/command/updatecmd.go.tmpl
	updateCmdTxt string

	//go:embed internal/client/dto/query/listbypageqry.go.tmpl
	listByPageQryTxt string

	//go:embed internal/client/dto/query/qry.go.tmpl
	qryTxt string

	//go:embed internal/client/viewobject/vo.go.tmpl
	voTxt string

	//go:embed internal/domain/model/model.go.tmpl
	modelTxt string

	//go:embed internal/domain/repository/repo.go.tmpl
	repoTxt string

	//go:embed internal/infra/dataobject/do.go.tmpl
	doTxt string

	//go:embed internal/infra/repository/cmdrepo.go.tmpl
	cmdRepoTxt string

	//go:embed internal/infra/repository/qryrepo.go.tmpl
	qryRepoTxt string
)

type CurdTemplate struct {
	Name      string
	Ext       string
	ParentDir string
	IsDir     bool
	Content   string
}

var CurdTemplates = []*CurdTemplate{
	{Name: "admin", ParentDir: "internal/adapter", IsDir: true},
	{Name: "controller", Ext: "go", ParentDir: "internal/adapter/admin", Content: controllerTxt},

	{Name: "createcmdexe", Ext: "go", ParentDir: "internal/app/executor/command", Content: createCmdExeTxt},
	{Name: "updatecmdexe", Ext: "go", ParentDir: "internal/app/executor/command", Content: updateCmdExeTxt},
	{Name: "deletecmdexe", Ext: "go", ParentDir: "internal/app/executor/command", Content: deleteCmdExeTxt},
	{Name: "getbyidqryexe", Ext: "go", ParentDir: "internal/app/executor/query", Content: getByIdQryExeTxt},
	{Name: "listbypageqryexe", Ext: "go", ParentDir: "internal/app/executor/query", Content: listByPageQryExeTxt},

	{Name: "createcmd", Ext: "go", ParentDir: "internal/client/dto/command", Content: createCmdTxt},
	{Name: "updatecmd", Ext: "go", ParentDir: "internal/client/dto/command", Content: updateCmdTxt},
	{Name: "listbypageqry", Ext: "go", ParentDir: "internal/client/dto/query", Content: listByPageQryTxt},
	{Name: "qry", Ext: "go", ParentDir: "internal/client/dto/query", Content: qryTxt},
	{Name: "vo", Ext: "go", ParentDir: "internal/client/viewobject", Content: voTxt},

	{Name: "model", Ext: "go", ParentDir: "internal/domain/model", Content: modelTxt},
	{Name: "repo", Ext: "go", ParentDir: "internal/domain/repository", Content: repoTxt},

	{Name: "do", Ext: "go", ParentDir: "internal/infra/dataobject", Content: doTxt},
	{Name: "cmdrepo", Ext: "go", ParentDir: "internal/infra/repository", Content: cmdRepoTxt},
	{Name: "qryrepo", Ext: "go", ParentDir: "internal/infra/repository", Content: qryRepoTxt},
}
