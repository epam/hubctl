package gcp

import "cloud.google.com/go/storage"

func IsNotFound(err error) bool {
	return err == storage.ErrObjectNotExist
}
