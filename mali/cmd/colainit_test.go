package cmd

import "testing"

func Test_runCola(t *testing.T) {
	type args struct {
		modDir string
		f      bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "cola-init",
			args: args{
				modDir: "./output/example",
				f:      true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCola(tt.args.modDir, tt.args.f)
		})
	}
}
