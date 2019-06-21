package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/pbkdf2"

	"hub/config"
)

const (
	encryptionMarkerByte0 = '\x26'
	encryptionMarkerByte1 = '\x01'
	encryptionSaltLen     = 8
	EncryptionOverhead    = 2 + encryptionSaltLen + 12 + 16 // marker salt nonce mac
)

var (
	encryptionKey  []byte
	encryptionSalt []byte
)

func IsEncryptedData(data []byte) bool {
	return data[0] == encryptionMarkerByte0 && data[1] == encryptionMarkerByte1
}

func encryptionKeyInit(salt []byte) []byte {
	if config.CryptoPassword == "" {
		log.Fatal("Set HUB_CRYPTO_PASSWORD='random password'")
	}
	return pbkdf2.Key([]byte(config.CryptoPassword), salt, 4096, 32, sha1.New)
}

func maybeEncryptionKeyInit() {
	if len(encryptionKey) == 0 {
		encryptionSalt = make([]byte, encryptionSaltLen)
		read, err := rand.Read(encryptionSalt)
		if err != nil {
			log.Fatalf("Unable to initialize encryption salt (read %d of %d random bytes): %v",
				read, len(encryptionSalt), err)
		}
		encryptionKey = encryptionKeyInit(encryptionSalt)
	}
}

func Encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	maybeEncryptionKeyInit()
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, data, encryptionSalt)
	buf := bytes.NewBuffer(make([]byte, 0, 2+len(encryptionSalt)+len(nonce)+len(ciphertext)))
	buf.WriteByte(encryptionMarkerByte0)
	buf.WriteByte(encryptionMarkerByte1)
	buf.Write(encryptionSalt)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

func Decrypt(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return encrypted, nil
	}
	if len(encrypted) < EncryptionOverhead+aes.BlockSize {
		return nil, errors.New("Insufficient ciphertext length")
	}
	if !IsEncryptedData(encrypted) {
		return nil, errors.New("Bad ciphertext marker")
	}
	encrypted = encrypted[2:]
	salt := encrypted[:encryptionSaltLen]
	rest := encrypted[encryptionSaltLen:]
	key := encryptionKeyInit(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce := rest[:nonceSize]
	ciphertext := rest[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, salt)
}

func Random(randomBytesLen int) (string, []byte, error) {
	buf := make([]byte, randomBytesLen)
	read, err := rand.Read(buf)
	if err != nil {
		return "", nil, fmt.Errorf("Unable to generate random chunk: random read error (read %d bytes): %v", read, err)
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
