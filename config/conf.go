package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Conf struct {
	AppEnv    AppEnv `yaml:"app_env"`
	AppName   string `yaml:"app_name"`
	filePath  string
	Workspace string                   `yaml:"workspace"`
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

func (c *Conf) GetWorkspace() string {
	if c.Workspace != "" {
		return c.Workspace
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		c.Workspace = os.TempDir() + "." + c.AppName
	} else {
		c.Workspace = dir + string(os.PathSeparator) + "." + c.AppName
	}

	if _, err = os.Stat(c.Workspace); err != nil {
		err = os.Mkdir(c.Workspace, 0755)
		if err != nil {
			fmt.Printf("创建工作目录[%s]失败:[%+v]", c.Workspace, err)
		}
	}

	return c.Workspace
}
