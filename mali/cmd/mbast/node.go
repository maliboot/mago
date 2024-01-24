package mbast

import (
	"strconv"
	"strings"

	"github.com/maliboot/mago/mali/cmd/mbast/attribute"
)

type Node struct {
	Name         string
	PackageName  string
	PackageAlias string
	PackageFQN   string
	Type         Type
	Receiver     *Node
	Attributes   []attribute.Attribute
}

type InterfaceNode = Node

type StructNode = Node

type BindNode = Node

func NewNodeFormString(name string, nodeType Type, nodeAttrs []attribute.Attribute) *Node {
	if !strings.Contains(name, "/") {
		return &Node{
			Name:         name,
			PackageName:  "",
			PackageAlias: "",
			PackageFQN:   "",
			Type:         0,
			Attributes:   nil,
		}
	}

	nameSlice := strings.Split(name, "/")
	nameSliceLen := len(nameSlice)
	return &Node{
		Name:         nameSlice[nameSliceLen-1],
		PackageName:  nameSlice[nameSliceLen-2],
		PackageAlias: "",
		PackageFQN:   strings.Join(nameSlice[:nameSliceLen-1], "/"),
		Type:         nodeType,
		Attributes:   nodeAttrs,
	}
}

func (n *Node) ResetAlias(nodes []*Node) *Node {
	pkgNames := make(map[string]int)
	nodeListLen := len(nodes)
	for i := 0; i < nodeListLen; i++ {
		if n.PackageFQN == nodes[i].PackageFQN {
			n.PackageAlias = nodes[i].PackageAlias
			return n
		}
		pkgNames[nodes[i].PackageName] = 1
	}

	if _, ok := pkgNames[n.PackageName]; ok {
		n.PackageAlias = n.PackageName + strconv.Itoa(nodeListLen+1)
	}
	return n
}

func (n *Node) GetTplRef() string {
	return n.GetTplRefPrefix() + "." + n.Name
}

func (n *Node) GetTplRefPrefix() string {
	if n.PackageAlias != "" {
		return n.PackageAlias
	}
	return n.PackageName
}
