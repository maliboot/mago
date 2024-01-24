package tpl

import (
	"os"

	"github.com/maliboot/mago/mali/cmd/mod"
	"github.com/maliboot/mago/mali/cmd/tpl/skeleton"
)

type ColaSkeletonTplArgs struct {
	ModName string
}

type ColaSkeleton struct {
	TplArgs *ColaSkeletonTplArgs
	mod     mod.Mod
	force   bool
}

func NewColaSkeleton(mod mod.Mod, force bool) *ColaSkeleton {
	return &ColaSkeleton{
		TplArgs: &ColaSkeletonTplArgs{ModName: mod.GetName()},
		mod:     mod,
		force:   force,
	}
}

func (cs *ColaSkeleton) Name() string {
	return "ColaSkeleton"
}

func (cs *ColaSkeleton) Initialize() {
}

func (cs *ColaSkeleton) Execute() error {
	modPath := cs.mod.GetPath()
	for _, t := range skeleton.Templates {
		p := modPath + "/" + t.Path
		if t.IsDir {
			_ = os.MkdirAll(p, 0755)
			continue
		}

		err := GenerateTpl(t.Content, cs, p, cs.force)
		if err != nil {
			return err
		}
	}

	return nil
}
