package tpl

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/mali/cmd/mbast"
	"github.com/maliboot/mago/mali/cmd/mod"
)

type Executor interface {
	Name() string
	Initialize()
	Execute() error
}

func GetInjectExecutors(mod mod.Mod, nodes mbast.Nodes) []Executor {
	return []Executor{
		NewWire(mod, nodes),
	}
}

func GetColaExecutors(mod mod.Mod, force bool) []Executor {
	return []Executor{
		NewColaSkeleton(mod, force),
	}
}

func GetColaCurdExecutors(mod mod.Mod, c *config.Conf, table string, force bool) []Executor {
	return []Executor{
		NewColaCurd(mod, c, table, force),
	}
}
