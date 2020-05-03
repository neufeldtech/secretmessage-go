package secretmessage

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func Test_encrypt_decrypt(t *testing.T) {
	type args struct {
		input      string
		passphrase string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Successful encryption and decryption",
			args: args{
				input:      "this is my secret",
				passphrase: "my passphrase",
			},
			want: "this is my secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encrypt(tt.args.input, tt.args.passphrase)
			got, err := decrypt(encrypted, tt.args.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("encrypt() or decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decrypt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hash(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return sha256 hash as hex",
			args: args{
				s: "my input string",
			},
			want: "9baecb53f4696b523d6de5c1e1942387383ecaf667c229602a12b79314a618e5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hash(tt.args.s); got != tt.want {
				t.Errorf("hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deriveCryptoKey(t *testing.T) {
	type args struct {
		key string
	}
	w, _ := hex.DecodeString("8af774dc91d8be77e2baccfb8856c103")
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "should return md5 as byte slice",
			args: args{
				key: "my input string",
			},
			want: w,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveCryptoKey(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deriveCryptoKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
