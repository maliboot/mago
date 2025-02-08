package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Conf struct {
	AppEnv    AppEnv `yaml:"app_env"`
	AppName   string `yaml:"app_name"`
	filePath  string
	Server    *ServerConf              `yaml:"server"`
	Log       *LogConf                 `yaml:"logger"`
	Databases map[string]*DataBaseConf `yaml:"databases"`
	File      *File                    `yaml:"file"`
	Redis     *Redis                   `yaml:"redis"`
	Ctx       any
}

func NewConf(opts ...ConfOption) *Conf {
	var c Conf
	for _, o := range opts {
		o(&c)
	}

	if c.AppEnv == "" {
		c.AppEnv = AppEnvFromStr(os.Getenv("APP_ENV"))
	}

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
	err, _ := c.Log.LoggerInit(c.AppEnv == Dev)
	if err != nil {
		return err
	}

	return nil
}
