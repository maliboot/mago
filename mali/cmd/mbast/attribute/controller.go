package attribute

type Controller struct {
	Prefix      string
	Middlewares []string
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
		if middlewares, ok := args["1"]; ok {
			c.Middlewares = formatMiddlewaresDoc(middlewares)
		}
		return c
	}

	if path, ok := args["prefix"]; ok {
		c.Prefix = path
	}
	if middlewares, ok := args["middlewares"]; ok {
		c.Middlewares = formatMiddlewaresDoc(middlewares)
	}
	return c
}
