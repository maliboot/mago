package mbast

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/maliboot/mago/mali/cmd/mbast/attribute"
)

type Doc string

func (d *Doc) ParseAttributes() []attribute.Attribute {
	pattern := `#\[([\w.]+)\(?(.*?)\)?\]`
	reg := regexp.MustCompile(pattern)
	regRes := reg.FindAllSubmatch([]byte(*d), -1)
	if regRes == nil {
		return nil
	}

	var result = make([]attribute.Attribute, 0)
	for i := 0; i < len(regRes); i++ {
		if attr := d.ParseAttribute(string(regRes[i][1]), string(regRes[i][2])); attr != nil {
			result = append(result, attr)
		}
	}
	return result
}

func (d *Doc) ParseAttribute(attrNameDoc string, attrArgsDoc string) attribute.Attribute {
	switch attrNameDoc {
	case "Dependency":
		return (&attribute.Dependency{}).InitArgs(d.parseAttributeArgs(attrArgsDoc))
	case "Inject":
		return (&attribute.Inject{}).InitArgs(d.parseAttributeArgs(attrArgsDoc))
	case "Config":
		return (&attribute.Conf{}).InitArgs(d.parseAttributeArgs(attrArgsDoc))
	case "Controller":
		return (&attribute.Controller{}).InitArgs(d.parseAttributeArgs(attrArgsDoc))
	case "RequestMapping":
		return (&attribute.RequestMapping{}).InitArgs(d.parseAttributeArgs(attrArgsDoc))
	}
	return nil
}

func (d *Doc) parseAttributeArgs(attrArgsDoc string) map[string]string {
	res := make(map[string]string)
	if attrArgsDoc == "" {
		return res
	}

	formatArg := func(arg string) string {
		var argVal = strings.TrimSpace(arg)
		if strings.Contains(argVal, "\"") {
			argVal = strings.Trim(argVal, "\"")
		}
		return argVal
	}

	argDocs := make([]string, 0)
	attrArgsDocLen := len(attrArgsDoc)
	splitCount := 0
	startIndex := 0
	for i := 0; i < attrArgsDocLen; i++ {
		if i+1 == attrArgsDocLen {
			argDocs = append(argDocs, attrArgsDoc[startIndex:])
			break
		}
		if attrArgsDoc[i] == '"' {
			splitCount++
			continue
		}
		if attrArgsDoc[i] == ',' && splitCount%2 == 0 {
			argDocs = append(argDocs, attrArgsDoc[startIndex:i])
			startIndex = i + 1
		}
	}

	for i, argDoc := range argDocs {
		var argName = strconv.Itoa(i)
		var argValue = formatArg(argDoc)

		if strings.Contains(argDoc, ":") {
			argDocSlice := strings.Split(argDoc, ":")
			argName = formatArg(argDocSlice[0])
			argValue = formatArg(argDocSlice[1])
		}

		res[argName] = argValue
	}

	return res
}
