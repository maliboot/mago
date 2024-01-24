package config

type ConfOption func(c *Conf)

func WithConfFile(path string) ConfOption {
	return func(c *Conf) {
		c.filePath = path
	}
}
