package attribute

import "strings"

type Attribute interface {
	Name() string
	FQN() string
	InitArgs(args map[string]string) Attribute
}

func formatMiddlewaresDoc(middlewaresDoc string) []string {
	var result = make([]string, 0)
	for _, middleware := range strings.Split(strings.ToUpper(middlewaresDoc), ",") {
		result = append(result, formatImport(middleware))
	}
	return result
}

func formatImport(importStr string) string {
	if strings.Contains(importStr, ".") {
		return strings.ReplaceAll(importStr, ".", "/")
	}

	return importStr
}
