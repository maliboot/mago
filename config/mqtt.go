package config

type MQTT struct {
	Ip       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Enable   bool   `yaml:"enable"`
}
