package config

type TcpConf struct {
	IP     string `yaml:"ip"`
	Port   int    `yaml:"port"`
	Enable bool   `yaml:"enable"`
}
