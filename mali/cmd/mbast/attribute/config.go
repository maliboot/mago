package attribute

type Conf struct {
}

func (c *Conf) Name() string {
	return "Config"
}

func (c *Conf) FQN() string {
	return "Config"
}

func (c *Conf) InitArgs(map[string]string) Attribute {
	return c
}
