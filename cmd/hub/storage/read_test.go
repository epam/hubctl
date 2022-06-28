package storage

import (
	"reflect"
	"testing"
)

func TestRemoteStoragePaths(t *testing.T) {
	type args struct {
		paths []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Should return nil", args{nil}, nil},
		{"Should return nil", args{[]string{}}, nil},
		{"Should return nil", args{[]string{"a"}}, nil},
		{"Should return a://a", args{[]string{"a", "a://a"}}, []string{"a://a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoteStoragePaths(tt.args.paths); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoteStoragePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkPath(t *testing.T) {
	type args struct {
		path string
		kind string
	}
	emptyFile := &File{Kind: "fs", Path: ""}
	tests := []struct {
		name    string
		args    args
		want    *File
		wantErr bool
	}{
		{"Should return pointer to file with kind fs and empty path", args{"", ""}, emptyFile, false},
		{"Should return pointer to file with kind fs and path", args{",", ""}, &File{Kind: "fs", Path: ","}, false},
		{"Should return pointer to file with kind fs and path", args{",", ""}, &File{Kind: "fs", Path: ","}, false},
		{"Should return pointer to file with kind s3 and path", args{"s3://a", "s3"}, &File{Kind: "s3", Path: "s3://a"}, false},
		{"Should return pointer to file with kind s3 and path", args{"s3://a,s3://b", "s3"}, &File{Kind: "s3", Path: "s3://a,s3://b"}, false},
		{"Should return pointer to file with kind gs and path", args{"gs://a", "gs"}, &File{Kind: "gs", Path: "gs://a"}, false},
		{"Should return pointer to file with kind s3 and path", args{"gs://a,gs://b", "gs"}, &File{Kind: "gs", Path: "gs://a,gs://b"}, false},
		{"Should return pointer to file with kind az and path", args{"az://a", "az"}, &File{Kind: "az", Path: "az://a"}, false},
		{"Should return pointer to file with kind s3 and path", args{"az://a,az://b", "az"}, &File{Kind: "az", Path: "az://a,az://b"}, false},
		{"Should return nil file and error", args{"hub://a", "hub"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPath(tt.args.path, tt.args.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	type args struct {
		paths []string
		kind  string
	}
	tests := []struct {
		name  string
		args  args
		want  *Files
		want1 []error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := Check(tt.args.paths, tt.args.kind)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Check() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Check() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestEnsureNoLockFiles(t *testing.T) {
	type args struct {
		files *Files
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			EnsureNoLockFiles(tt.args.files)
		})
	}
}

func Test_chooseFile(t *testing.T) {
	type args struct {
		files *Files
	}
	tests := []struct {
		name    string
		args    args
		want    *File
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chooseFile(tt.args.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("chooseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chooseFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readFile(t *testing.T) {
	type args struct {
		file *File
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readFile(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chooseAndReadFile(t *testing.T) {
	type args struct {
		files *Files
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := chooseAndReadFile(tt.args.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("chooseAndReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chooseAndReadFile() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("chooseAndReadFile() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestRead(t *testing.T) {
	type args struct {
		files *Files
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := Read(tt.args.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Read() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Read() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCheckAndRead(t *testing.T) {
	type args struct {
		paths []string
		kind  string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := CheckAndRead(tt.args.paths, tt.args.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAndRead() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckAndRead() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("CheckAndRead() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
