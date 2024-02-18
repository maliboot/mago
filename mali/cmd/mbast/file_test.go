package mbast

import (
	"github.com/maliboot/mago/mali/cmd/mod"
	"go/ast"
	"testing"
)

func TestFile_getPkgName(t *testing.T) {
	type fields struct {
		path string
		mod  mod.Mod
		ast  *ast.File
	}
	myMod := mod.NewMod("../../../")
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "windows-file-success",
			fields: fields{
				path: "C:\\Users\\Administrator\\Code\\mago\\config\\conf.go",
				mod:  myMod,
			},
			want: "config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				path: tt.fields.path,
				mod:  tt.fields.mod,
				ast:  tt.fields.ast,
			}
			if got := f.getPkgName(); got != tt.want {
				t.Errorf("getPkgName() = %v, want %v", got, tt.want)
			}
		})
	}
}
