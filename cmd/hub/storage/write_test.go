package storage

import (
	"errors"
	"os"
	"testing"

	"github.com/agilestacks/hub/cmd/hub/config"
)

func getRealFile() *Files {
	f, _ := os.CreateTemp("", "hub_storage_write_test")
	realFiles, _ := Check([]string{f.Name()}, "real kind")

	return realFiles
}

func TestWrite(t *testing.T) {
	type args struct {
		data  []byte
		files *Files
		encrypted bool
		compressed bool
	}
	fakeFiles, _ := Check([]string{"/a/b/c/d"}, "fake kind")

	tests := []struct {
		name  string
		args  args
		want  bool
		wantErr []error
	}{
		{
			"Should write to file without encryption and compression",
			args{
				[]byte("This is test data"),
				getRealFile(),
				false, false,
			},
			true,
			nil,
		},
		{
			"Should write to file with encryption and without compression",
			args{
				[]byte("This is test data"),
				getRealFile(),
				true, false,
			},
			true,
			nil,
		},
		{
			"Should write to file without encryption and with compression",
			args{
				[]byte("This is test data"),
				getRealFile(),
				false, true,
			},
			true,
			nil,
		},
		{
			"Should write to file with encryption and compression",
			args{
				[]byte("This is test data"),
				getRealFile(),
				true, true,
			},
			true,
			nil,
		},
		{"Should return false", args{[]byte("This is test data"), fakeFiles, false, false}, false, []error{errors.New("Unable to open `/a/b/c/d` fake kind file for write: open /a/b/c/d: no such file or directory")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Encrypted = tt.args.encrypted
			config.Compressed = tt.args.compressed
			got, err := Write(tt.args.data, tt.args.files)
			if len(err) != len(tt.wantErr) {
				t.Errorf("Write() err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Write() got = %v, want %v", got, tt.want)
			}
		})
	}
}
