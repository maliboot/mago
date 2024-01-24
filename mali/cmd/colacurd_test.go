package cmd

import "testing"

func Test_runColaCurd(t *testing.T) {
	type args struct {
		modDir  string
		dbTable string
		f       bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "cola-curd",
			args: args{
				modDir:  "./output/example",
				dbTable: "uss_message_tpl_var",
				f:       true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runColaCurd(tt.args.modDir, tt.args.dbTable, tt.args.f)
		})
	}
}
