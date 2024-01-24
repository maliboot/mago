package attribute

type Attribute interface {
	Name() string
	FQN() string
	InitArgs(args map[string]string) Attribute
}
