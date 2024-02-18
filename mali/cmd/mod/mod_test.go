package mod

import "testing"

func Test_mod_GetPkgFQN(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		filePath string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "windows-mod-success",
			fields: fields{
				path: "../../../",
			},
			args: args{filePath: "C:\\Users\\Administrator\\Code\\mago\\config\\conf.go"},
			want: "github.com/maliboot/mago/config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMod(tt.fields.path)
			if got := m.GetPkgFQN(tt.args.filePath); got != tt.want {
				t.Errorf("GetPkgFQN() = %v, want %v", got, tt.want)
			}
		})
	}
}
