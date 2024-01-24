package mbast

import (
	"strconv"

	"github.com/samber/lo"
)

type Nodes []*Node

func (l Nodes) FillNodesAlias() Nodes {
	return l.fillAlias()
}

func (l Nodes) fillAlias() Nodes {
	var aliasMap = make(map[string]string)

	for i := 0; i < len(l); i++ {
		pkgFQN := l[i].PackageFQN
		if i == 0 {
			aliasMap[pkgFQN] = l[i].PackageName
			continue
		}

		// exist, not log
		if alias, ok := aliasMap[pkgFQN]; ok {
			if alias != l[i].PackageName {
				l[i].PackageAlias = alias
			}
			continue
		}

		// not exist, depend on
		if len(lo.PickBy(aliasMap, func(key string, value string) bool { return l[i].PackageName == value })) != 0 {
			pkgAlias := l[i].PackageName + strconv.Itoa(i)
			l[i].PackageAlias = pkgAlias
			aliasMap[pkgFQN] = pkgAlias
			continue
		}

		aliasMap[pkgFQN] = l[i].PackageName
	}
	return l
}

func (l Nodes) GetAlisaFQNs() map[string]string {
	res := make(map[string]string)

	for i := 0; i < len(l); i++ {
		res[l[i].GetTplRef()] = l[i].PackageFQN
	}

	return res
}
