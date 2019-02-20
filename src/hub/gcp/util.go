package gcp

func IsNotFound(err error) bool {
	str := err.Error()
	return str == "storage: object doesn't exist"
}
