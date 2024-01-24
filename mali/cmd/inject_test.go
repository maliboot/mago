package cmd

import "testing"

func Test_runInject(t *testing.T) {
	type args struct {
		workDir string
		workNS  string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "cmd-inject:success",
			args: args{
				workDir: "./output/example",
				workNS:  "example",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInject(tt.args.workDir)
		})
	}
}
