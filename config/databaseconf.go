package config

import (
	"errors"
	"sync"

	"github.com/maliboot/mago/log"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DataBaseConf struct {
	Type          string `yaml:"type"`
	Dsn           string `yaml:"dsn"`
	TablePrefix   string `yaml:"table_prefix"`
	SingularTable bool   `yaml:"singular_table"`
	NoLowerCase   bool   `yaml:"no_lower_case"`
	dbConnector   *dbConnector
	Ctx           any
}

type dbConnector struct {
	ins *gorm.DB
	sync.Once
	err error
}

func (dc *DataBaseConf) initDbConnector() {
	if dc.dbConnector == nil {
		dc.dbConnector = &dbConnector{}
	}
	if dc.Type == "" {
		dc.Type = "mysql"
	}

	var dialector gorm.Dialector
	switch dc.Type {
	case "mysql":
		dialector = mysql.New(mysql.Config{DSN: dc.Dsn})
		break
	case "sqlite":
		dialector = sqlite.Open(dc.Dsn)
		break
	default:
		dc.dbConnector.err = errors.New("不支持的数据库类型")
		return
	}

	dc.dbConnector.Do(func() {
		dc.dbConnector.ins, dc.dbConnector.err = gorm.Open(dialector, &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   dc.TablePrefix,
				SingularTable: dc.SingularTable,
				NoLowerCase:   dc.NoLowerCase,
			},
			Logger: log.NewGormLogger(),
		})
	})
}

func (dc *DataBaseConf) GetDB() (*gorm.DB, error) {
	dc.initDbConnector()
	return dc.dbConnector.ins, dc.dbConnector.err
}
