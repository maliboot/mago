package config

type ServerConf struct {
	Http HttpConf `yaml:"http"`
	Tcp  TcpConf  `yaml:"tcp"`
	Udp  UdpConf  `yaml:"udp"`
	Ctx  any
}
