package tpl

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/mali/cmd/mod"
	"github.com/maliboot/mago/mali/cmd/tpl/skeleton"
)

type ColaCurdTableField struct {
	Name           string
	LowerCamelName string
	UpperCamelName string
	Type           string
}

type ColaCurdTplArgs struct {
	ModName             string
	ModDelimitedName    string
	DBName              string
	TableName           string
	LowerTableName      string
	LowerCamelTableName string
	UpperCamelTableName string
	TableFields         []*ColaCurdTableField
}

type ColaCurd struct {
	TplArgs   *ColaCurdTplArgs
	mod       mod.Mod
	force     bool
	dbName    string
	tableName string
	c         *config.Conf
}

func NewColaCurd(mod mod.Mod, c *config.Conf, table string, force bool) *ColaCurd {
	return &ColaCurd{
		TplArgs: &ColaCurdTplArgs{
			ModName:          mod.GetName(),
			ModDelimitedName: strings.ReplaceAll(mod.GetName(), "/", "."),
			TableFields:      make([]*ColaCurdTableField, 0),
		},
		mod:       mod,
		force:     force,
		tableName: table,
		c:         c,
	}
}

func (cs *ColaCurd) Name() string {
	return "ColaCurd"
}

func (cs *ColaCurd) Initialize() {
	if strings.Contains(cs.tableName, ".") {
		tSlice := strings.Split(cs.tableName, ".")
		tSliceLen := len(tSlice)
		if tSliceLen < 1 {
			return
		} else if tSliceLen == 1 {
			cs.tableName = tSlice[0]
		} else {
			cs.dbName = tSlice[0]
			cs.tableName = tSlice[1]
		}
	}
	if len(cs.c.Databases) == 0 {
		panic(fmt.Sprintf("[conf.yml]没有数据库配置"))
	}
	if cs.dbName == "" {
		for dbKey := range cs.c.Databases {
			cs.dbName = dbKey
			break
		}
	}
	cs.TplArgs.TableName = cs.tableName
	cs.TplArgs.LowerCamelTableName = strcase.ToLowerCamel(cs.tableName)
	cs.TplArgs.UpperCamelTableName = strcase.ToCamel(cs.tableName)
	cs.TplArgs.LowerTableName = strings.ToLower(cs.TplArgs.LowerCamelTableName)
	cs.TplArgs.DBName = cs.dbName

	// table fields
	db, err := cs.c.Databases[cs.dbName].GetDB()
	if err != nil {
		panic(fmt.Sprintf("获取数据库链接异常:%v", err))
	}

	mi := db.Migrator()
	if !mi.HasTable(cs.tableName) {
		panic(fmt.Sprintf("数据库[%s]内表[%s]不存在", cs.dbName, cs.tableName))
	}

	rows, err := db.Table(cs.tableName).Rows()
	if err != nil {
		panic(fmt.Sprintf("获取表字段信息异常:%v", err))
	}
	columns, err := rows.ColumnTypes()
	if err != nil {
		panic(fmt.Sprintf("获取表字段类型信息异常:%v", err))
	}

	for _, column := range columns {
		gType := "any"
		dbType := column.DatabaseTypeName()
		if strings.Contains(dbType, "INT") {
			gType = "int"
		}
		if strings.Contains(dbType, "VARCHAR") || strings.Contains(dbType, "JSON") || strings.Contains(dbType, "TEXT") {
			gType = "string"
		}
		if strings.Contains(dbType, "TIMESTAMP") {
			gType = "mago.DateTime"
		}

		cs.TplArgs.TableFields = append(cs.TplArgs.TableFields, &ColaCurdTableField{
			Name:           column.Name(),
			LowerCamelName: strcase.ToLowerCamel(column.Name()),
			UpperCamelName: strcase.ToCamel(column.Name()),
			Type:           gType,
		})
	}
}

func (cs *ColaCurd) Execute() error {
	modPath := cs.mod.GetPath()
	prefix := cs.TplArgs.LowerTableName

	// 模型目录
	tpls := slices.Insert(skeleton.CurdTemplates, 0, &skeleton.CurdTemplate{Name: prefix, ParentDir: "internal/domain/model", IsDir: true})

	for _, t := range tpls {
		p := modPath + "/" + t.ParentDir
		if t.IsDir {
			_ = os.MkdirAll(p+"/"+t.Name, 0755)
			continue
		}

		switch t.Name {
		case "model":
			p = fmt.Sprintf("%s/%s/%s.%s", p, prefix, prefix, t.Ext)
			break
		default:
			p = fmt.Sprintf("%s/%s%s.%s", p, prefix, t.Name, t.Ext)
		}

		err := GenerateTpl(t.Content, cs, p, cs.force)
		if err != nil {
			return err
		}
	}

	return nil
}
