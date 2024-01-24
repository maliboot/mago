package tpl

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

func GenerateTpl(text string, data any, targetPath string, force bool) error {
	_, err := os.Stat(targetPath)
	if err == nil {
		if !force {
			fmt.Printf("%s文件已存在，不再创建\n", targetPath)
			return nil
		}
		err = ForceGenerateTpl(text, data, targetPath)
		if err == nil {
			fmt.Printf("%s强制覆盖成功\n", targetPath)
		}
		return err
	}

	err = ForceGenerateTpl(text, data, targetPath)
	if err == nil {
		fmt.Printf("%s创建成功\n", targetPath)
	}
	return err
}

func ForceGenerateTpl(text string, data any, targetPath string) error {
	var tplText bytes.Buffer
	var targetTplS = strings.Split(targetPath, "/")
	tpl := template.Must(template.New(targetTplS[len(targetTplS)-1]).Parse(text))
	err := tpl.Execute(&tplText, data)
	if err != nil {
		return err
	}

	return GenerateFile(targetPath, tplText.Bytes())
}

func GenerateFile(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	_, err = f.Write(content)
	if err != nil {
		_ = f.Close()
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}
