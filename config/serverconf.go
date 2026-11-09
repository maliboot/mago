package config

type ServerConf struct {
	Http HttpConf `yaml:"http"`
	Tcp  TcpConf  `yaml:"tcp"`
	Udp  UdpConf  `yaml:"udp"`

	ServerMachineId string `yaml:"server_machine_id"`
	MemoryLimit     uint64 `yaml:"memory_limit"` // MB
	CoroutineLimit  uint64 `yaml:"coroutine_limit"`
	Ctx             any
}
