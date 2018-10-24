package storage

import (
	"time"
)

type File struct {
	Kind    string
	Path    string
	Exist   bool
	Size    int64
	ModTime time.Time
	Locked  bool
}

type Files struct {
	Kind  string
	Files []File
}
