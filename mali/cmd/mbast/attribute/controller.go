package attribute

type Controller struct {
	Prefix string
}

func (c *Controller) Name() string {
	return "Controller"
}

func (c *Controller) FQN() string {
	return "Controller"
}

func (c *Controller) InitArgs(args map[string]string) Attribute {
	if path, ok := args["0"]; ok {
		c.Prefix = path
		return c
	}

	if path, ok := args["Prefix"]; ok {
		c.Prefix = path
	}
	return c
}
