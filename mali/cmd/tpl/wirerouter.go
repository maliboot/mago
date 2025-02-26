package tpl

type WireMethods struct {
	Name            string
	Path            string
	Methods         []string
	MiddlewaresPack string
}

type WireController struct {
	Name    string
	Uniqid  string
	Methods map[string]*WireMethods
}
