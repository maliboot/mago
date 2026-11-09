package helper

import "testing"

func TestTelnetLikeTest(t *testing.T) {
	type args1 struct {
		ip   string
		port string
	}
	//117.62.242.215
	//180.111.198.104
	//183.60.230.109
	//183.2.144.19
	//180.102.211.178
	//114.221.151.169
	//180.102.211.178
	//114.221.151.169
	tests := []struct {
		name    string
		args    args1
		wantErr bool
	}{
		{
			args: args1{
				ip:   "180.102.48.188",
				port: "443",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := TelnetLikeTest(tt.args.ip, tt.args.port); (err != nil) != tt.wantErr {
				t.Errorf("TelnetLikeTest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
