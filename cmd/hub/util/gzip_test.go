package util

import (
	"reflect"
	"testing"
)

const EMPTY_GZIP = "\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00"
const TEST_STRING = "Test string"
const GZIPED_TEST_STRING = "\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\nI-.Q(.)\xca\xccK\a\x04\x00\x00\xff\xff\x92\x9aە\v\x00\x00\x00"

func TestGzip(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"Should return empty gzip bytes", args{nil}, []byte(EMPTY_GZIP), false},
		{"Should return empty gzip bytes", args{[]byte{}}, []byte(EMPTY_GZIP), false},
		{"Should return gziped bytes", args{[]byte(TEST_STRING)}, []byte(GZIPED_TEST_STRING), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Gzip(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Gzip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Gzip() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsGzipData(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should return false for nil", args{nil}, false},
		{"Should return false for empty byte array", args{[]byte{}}, false},
		{"Should return true for empty gzip", args{[]byte(EMPTY_GZIP)}, true},
		{"Should return true for gzip string", args{[]byte(GZIPED_TEST_STRING)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGzipData(tt.args.data); got != tt.want {
				t.Errorf("IsGzipData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGunzip(t *testing.T) {
	type args struct {
		compressed []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"Should return empty byte array", args{[]byte(EMPTY_GZIP)}, []byte(""), false},
		{"Should return error", args{nil}, nil, true},
		{"Should return test string byte array", args{[]byte(GZIPED_TEST_STRING)}, []byte(TEST_STRING), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Gunzip(tt.args.compressed)
			if (err != nil) != tt.wantErr {
				t.Errorf("Gunzip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Gunzip() = %v, want %v", got, tt.want)
			}
		})
	}
}
