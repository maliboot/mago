package tpl

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/maliboot/mago/mali/cmd/mbast"
	"github.com/maliboot/mago/mali/cmd/mbast/attribute"
	"github.com/maliboot/mago/mali/cmd/mod"
)

//go:embed wire.tmpl
var wireTplTxt string

type WireAutoloadFunc struct {
	Name       string
	LcName     string
	RefName    string
	InjectFunc string
}

type WireTplArgs struct {
	Imports       map[string]string
	BindStructs   map[string]string
	InjectsFuncs  []string
	AutoloadFuncs map[string]*WireAutoloadFunc
	Routers       map[string]*WireController
}

type Wire struct {
	mod     mod.Mod
	tplText string
	nodes   mbast.Nodes
	TplArgs *WireTplArgs
}

func NewWire(modIns mod.Mod, nodes mbast.Nodes) Executor {
	tplArgs := &WireTplArgs{
		Imports:       make(map[string]string),
		BindStructs:   make(map[string]string),
		InjectsFuncs:  make([]string, 0),
		AutoloadFuncs: make(map[string]*WireAutoloadFunc),
		Routers:       make(map[string]*WireController),
	}

	ins := &Wire{mod: modIns, tplText: wireTplTxt, nodes: nodes, TplArgs: tplArgs}
	return ins
}

func (w *Wire) Name() string {
	return "wire"
}

func (w *Wire) Initialize() {
	w.nodes = w.nodes.FillNodesAlias()

	//swapImports := w.nodes.GetAlisaFQNs()
	for _, node := range w.nodes {
		nodeRef := node.GetTplRef()
		if node.Attributes == nil || len(node.Attributes) == 0 {
			continue
		}

		for _, attr := range node.Attributes {
			switch attr.(type) {
			case *attribute.Dependency:
				impl := attr.(*attribute.Dependency).Impl
				if impl != "" {
					implNode := mbast.NewNodeFormString(impl, mbast.StructType, nil)
					implNode.ResetAlias(w.nodes)

					w.TplArgs.Imports[node.PackageFQN] = node.PackageAlias
					if _, ok := w.TplArgs.Imports[implNode.PackageFQN]; !ok {
						w.TplArgs.Imports[implNode.PackageFQN] = implNode.PackageAlias
					}
					w.TplArgs.BindStructs[nodeRef] = implNode.GetTplRef()
					w.nodes = append(w.nodes, implNode) // 防止包名重复
				}
			case *attribute.Inject:
				w.TplArgs.Imports[node.PackageFQN] = node.PackageAlias
				if !slices.Contains(w.TplArgs.InjectsFuncs, nodeRef) {
					w.TplArgs.InjectsFuncs = append(w.TplArgs.InjectsFuncs, nodeRef)
				}
			case *attribute.Conf:
				w.TplArgs.Imports[node.PackageFQN] = node.PackageAlias
				if _, ok := w.TplArgs.AutoloadFuncs[node.Name]; !ok {
					nodeLcName := strings.ToLower(node.Name[0:1]) + node.Name[1:]
					w.TplArgs.AutoloadFuncs[node.Name] = &WireAutoloadFunc{
						Name:       node.Name,
						LcName:     nodeLcName,
						RefName:    node.GetTplRefPrefix(),
						InjectFunc: "New" + node.Name + "Conf",
					}
				}
			case *attribute.Controller:
				w.TplArgs.Imports[node.PackageFQN] = node.PackageAlias

			case *attribute.RequestMapping:
				if node.Receiver == nil || len(node.Receiver.Attributes) == 0 {
					continue
				}

				var rController *attribute.Controller
				for _, receiverAttr := range node.Receiver.Attributes {
					if rc, ok := receiverAttr.(*attribute.Controller); ok {
						rController = rc
					}
				}
				if rController == nil {
					continue
				}

				w.TplArgs.Imports[node.PackageFQN] = node.PackageAlias

				nReceiverRef := node.Receiver.GetTplRef()
				if _, ok := w.TplArgs.Routers[nReceiverRef]; !ok {
					w.TplArgs.Routers[nReceiverRef] = &WireController{
						Name:    node.Receiver.Name,
						Uniqid:  strings.ReplaceAll(nReceiverRef, ".", ""),
						Methods: make(map[string]*WireMethods),
					}
				}

				rmAttr := attr.(*attribute.RequestMapping)
				finalPath := rmAttr.GetPathByPrefix(rController.Prefix)
				w.TplArgs.Routers[nReceiverRef].Methods[nodeRef] = &WireMethods{
					Name:    node.Name,
					Path:    finalPath,
					Methods: rmAttr.Methods,
				}
			}
		}
	}

	// 基本空间导入
	if len(w.TplArgs.AutoloadFuncs) != 0 {
		w.TplArgs.Imports["github.com/maliboot/mago/config"] = "_mbconf"
		w.TplArgs.Imports["fmt"] = ""
	}
}

func (w *Wire) Execute() error {
	if w.isEmpty() {
		return nil
	}

	fmt.Printf(
		"%s扫描结果: Config[%d]个, Dependency[%d]个，Inject[%d]个",
		w.Name(),
		len(w.TplArgs.AutoloadFuncs),
		len(w.TplArgs.BindStructs),
		len(w.TplArgs.InjectsFuncs),
	)

	return ForceGenerateTpl(w.tplText, w, w.mod.GetPath()+"/provider_gen.go")
}

func (w *Wire) isEmpty() bool {
	return len(w.TplArgs.BindStructs) == 0 && len(w.TplArgs.InjectsFuncs) == 0 && len(w.TplArgs.AutoloadFuncs) == 0
}
