package secretmessage

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

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

func Test_secureSecretID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return argon2 hash as hex",
			args: args{
				s: "my input string",
			},
			want: "268aa12ad29ed592c4e73e727fc2152bb5f1edf343c61a9f5bf00d0c14b0a572",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secureSecretID(tt.args.s); got != tt.want {
				t.Errorf("secureSecretID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deriveCryptoKey(t *testing.T) {
	type args struct {
		key string
	}
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
			want: []byte{138, 247, 116, 220, 145, 216, 190, 119, 226, 186, 204, 251, 136, 86, 193, 3},
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
func Test_deriveCryptoKeyV2(t *testing.T) {
	type args struct {
		key  string
		salt string
	}

	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "should return argon2 byte slice",
			args: args{
				key:  "my input string",
				salt: "DZNUVLZNJVR3HOWSZPM2DEEKQ3",
			},
			want: []byte{86, 190, 230, 34, 101, 60, 153, 105, 175, 164, 186, 225, 142, 50, 228, 245, 103, 115, 222, 104, 57, 160, 91, 32, 64, 134, 165, 228, 139, 67, 11, 173},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveCryptoKeyV2(tt.args.key, tt.args.salt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deriveCryptoKeyV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_encryptWithReader(t *testing.T) {

	type args struct {
		rr         io.Reader
		input      string
		passphrase string
		salt       string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "successful encryption with salt",
			args: args{
				rr:         bytes.NewReader([]byte("00000000000000000000000000000000")),
				input:      "the password is baseball123",
				passphrase: "monkey",
				salt:       "VC4TZT7JZOAAVFQ3F3N7GXF2RP",
			},
			want: "v2$VC4TZT7JZOAAVFQ3F3N7GXF2RP$303030303030303030303030a091fd029ae527dfa9ed207ba09c537d8bdb5012f264f2a65d68a333fd58b378e5791a94d3060e919d4486",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encryptWithReader(tt.args.rr, tt.args.input, tt.args.passphrase, tt.args.salt)
			if (err != nil) != tt.wantErr {
				t.Errorf("encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("encrypt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decrypt(t *testing.T) {
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
			name: "successful decryption V1",
			args: args{
				input:      "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585",
				passphrase: "monkey",
			},
			want: "the password is baseball123",
		},
		{
			name: "successful decryption V2",
			args: args{
				input:      "v2$VC4TZT7JZOAAVFQ3F3N7GXF2RP$303030303030303030303030a091fd029ae527dfa9ed207ba09c537d8bdb5012f264f2a65d68a333fd58b378e5791a94d3060e919d4486",
				passphrase: "monkey",
			},
			want: "the password is baseball123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decrypt(tt.args.input, tt.args.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decrypt() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
