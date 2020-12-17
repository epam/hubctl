package util

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func Random(randomBytesLen int) (string, []byte, error) {
	buf := make([]byte, randomBytesLen)
	read, err := rand.Read(buf)
	if err != nil {
		return "", nil, fmt.Errorf("Unable to generate random chunk: random read error (read %d of %d random bytes): %v",
			read, randomBytesLen, err)
	}
	return base64.RawStdEncoding.EncodeToString(buf), buf, nil
}

func OtpEncode(input []byte, random []byte) (string, error) {
	if len(input) > len(random) {
		return "", fmt.Errorf("Input length of %d bytes is larger than one-time pad length of %d bytes", len(input), len(random))
	}
	if len(input) < len(random) {
		bytes2 := make([]byte, len(random))
		copy(bytes2, input)
		input = bytes2
	}
	for i := 0; i < len(input); i++ {
		input[i] ^= random[i]
	}
	return base64.RawStdEncoding.EncodeToString(input), nil
}

func OtpDecode(base64Input string, random []byte) ([]byte, error) {
	input, err := base64.RawStdEncoding.DecodeString(base64Input)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode base64 input: %v", err)
	}
	if len(input) != len(random) {
		return nil, fmt.Errorf("Input length of %d bytes is not equal to one-time pad length of %d bytes", len(input), len(random))
	}
	for i := 0; i < len(input); i++ {
		input[i] ^= random[i]
	}
	i := bytes.IndexByte(input, 0)
	if i > -1 {
		input = input[:i]
	}
	return input, nil
}
