package attribute

type Dependency struct {
	Impl string
}

func (d *Dependency) Name() string {
	return "Dependency"
}

func (d *Dependency) FQN() string {
	return "Dependency"
}

func (d *Dependency) InitArgs(args map[string]string) Attribute {
	if impl, ok := args["0"]; ok {
		d.Impl = formatImport(impl)
		return d
	}

	if impl, ok := args["impl"]; ok {
		d.Impl = formatImport(impl)
	}
	return d
}
