package mask

import (
	"encoding/json"
	"reflect"
	"testing"
)

var desIns, _ = New3Des("CwCv4fV1N8sufNOwrVxi00tBz4Kxqabl")

var plain, _ = json.Marshal(map[string]interface{}{
	"applicantType": 1,
	"certID":        "22040219790724145X",
	"certType":      "1010",
	"custName":      "stone",
	"isInterval":    "0",
	"legalName":     "stone",
	"legalType":     "1010",
	"mobile":        "87601023200",
	"orgCode":       "",
	"orgType":       "02",
})

var crypted = "NWU730cE+Hd4+i2VBXYK/E2QRqjwKl/9x6TMK536QB5qvnBrgzj01JLqUHABYKAYpu8AAm1LSDFobo7xkoUIe2jUB4d+IDDHm05k0Gvfh1qr9fUGerlMx0ItRMUnlqMvBvQ2XcEEr3JcOlB9OXWOGxYxPC8bkQUjbEUd5Q53RI5x7zYOuLXtcXD2EyNGUvjYmnW6yKpfW11PrjmOxLd1SLDaWvqZ/WX9yk5L1flPVuIXrkaRjJw5zfw+id4MAwb0Qsec6qmcCnc="

func Test_tripleDes_Encrypt(t *testing.T) {
	type fields struct {
	}
	type args struct {
		plaintext []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:   "encrypt-0",
			fields: fields{},
			args:   args{plaintext: plain},
			want:   crypted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := desIns
			got, err := td.EcbEncrypt(tt.args.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encrypt() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tripleDes_Decrypt(t *testing.T) {
	type fields struct {
	}
	type args struct {
		ciphertextBase64En string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:   "decrpyt-0",
			fields: fields{},
			args:   args{ciphertextBase64En: crypted},
			want:   plain,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := desIns
			got, err := td.EcbDecrypt(tt.args.ciphertextBase64En)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decrypt() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tripleDes_RandomSecretKey(t *testing.T) {
	type fields struct {
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "3des-key-0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := desIns
			got, err := td.RandomSecretKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("RandomSecretKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("RandomSecretKey() got = %v", got)
		})
	}
}
