package config

type UdpConf struct {
	IP     string `yaml:"ip"`
	Port   int    `yaml:"port"`
	Enable bool   `yaml:"enable"`
}
