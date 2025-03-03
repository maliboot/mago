package config

type HttpConf struct {
	IP   string              `yaml:"ip"`
	Port int                 `yaml:"port"`
	JWT  map[string]*JWTConf `yaml:"jwt"`
}
