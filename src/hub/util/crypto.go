package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"errors"
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
