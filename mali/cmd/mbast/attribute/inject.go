package attribute

type Inject struct {
}

func (i *Inject) Name() string {
	return "Inject"
}

func (i *Inject) FQN() string {
	return "Inject"
}

func (i *Inject) InitArgs(args map[string]string) Attribute {
	return i
}
