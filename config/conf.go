package config

import (
	"fmt"
	"github.com/maliboot/mago/logx"
	"io"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type Conf struct {
	AppEnv    AppEnv `yaml:"app_env"`
	AppName   string `yaml:"app_name"`
	filePath  string
	Workspace string                   `yaml:"workspace"`
	Server    *ServerConf              `yaml:"server"`
	Log       *LogRusConf              `yaml:"logger"`
	Databases map[string]*DataBaseConf `yaml:"databases"`
	File      *File                    `yaml:"file"`
	Redis     *Redis                   `yaml:"redis"`
	MQTT      *MQTT                    `yaml:"mqtt"`
	SMTP      *SMTP                    `yaml:"smtp"`
	Queue     *QueueConf               `yaml:"queue"`
	Scheduler *Scheduler               `yaml:"scheduler"`
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
		c.Log = &LogRusConf{}
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

func (c *Conf) Bootstrap(fixedProduction bool) error {
	// 配置解析
	if c.filePath != "" {
		err := c.fileRead(c)
		if err != nil {
			return err
		}
	}
	if fixedProduction {
		c.AppEnv = Production
	}

	// 日志
	logx.NewLogRus(c.AppEnv.String(), c.Log.LogDir)

	// 队列
	if c.Queue == nil {
		c.Queue = &QueueConf{}
	}
	if c.Queue.CacheDir == "" {
		c.Queue.CacheDir = path.Join(c.Log.LogDir, "queue")
	}

	// 计划任务
	if c.Scheduler == nil {
		c.Scheduler = &Scheduler{}
		cron, err := c.Scheduler.Cron()
		if err != nil {
			return err
		}
		cron.Start()
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
