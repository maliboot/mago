package config

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Conf struct {
	AppEnv    AppEnv
	AppName   string `yaml:"app_name"`
	filePath  string
	Server    *ServerConf              `yaml:"server"`
	Log       *LogConf                 `yaml:"logger"`
	Databases map[string]*DataBaseConf `yaml:"databases"`
}

func NewConf(opts ...ConfOption) *Conf {
	var c Conf
	for _, o := range opts {
		o(&c)
	}

	c.AppEnv = AppEnvFromStr(strings.ToLower(os.Getenv("APP_ENV")))
	if c.filePath == "" {
		c.Server = &ServerConf{
			Http: HttpConf{Port: 9501},
		}
		c.Log = &LogConf{}
	}

	return &c
}

func (c *Conf) Scan(out interface{}) error {
	if c.filePath == "" {
		return fmt.Errorf("config.yml was not fund")
	}

	err := c.fileRead(out)
	if err != nil {
		return err
	}
	return nil
}

func (c *Conf) fileRead(out interface{}) error {
	f, err := os.Open(c.filePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	bytes, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("读取配置文件错误:%v", err.Error())
	}
	err = yaml.Unmarshal(bytes, out)
	if err != nil {
		return fmt.Errorf("配置结构体解析错误:%v", err.Error())
	}
	return nil
}

func (c *Conf) Bootstrap() error {
	// 配置解析
	if c.filePath != "" {
		err := c.fileRead(c)
		if err != nil {
			return err
		}
	}

	// 日志
	err, _ := c.Log.LoggerInit()
	if err != nil {
		return err
	}

	return nil
}
