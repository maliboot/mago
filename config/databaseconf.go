package config

import (
	"sync"

	"github.com/maliboot/mago/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DataBaseConf struct {
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
	dc.dbConnector.Do(func() {
		dc.dbConnector.ins, dc.dbConnector.err = gorm.Open(mysql.New(mysql.Config{
			DSN: dc.Dsn,
		}), &gorm.Config{
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
