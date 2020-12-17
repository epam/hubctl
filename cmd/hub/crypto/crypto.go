package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"

	"github.com/agilestacks/hub/cmd/hub/aws"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

const (
	encryptionMarkerByte0        = '\x26'
	encryptionV1MarkerByte1      = '\x01'
	encryptionV2MarkerByte1      = '\x02'
	encryptionV1SaltLen          = 8
	encryptionNonceLen           = 12
	encryptionV2EncryptedBlobLen = 184 // AES256 key and 152 bytes of fixed-size AWS KMS meta
	encryptionMacLen             = 16

	EncryptionV1Overhead = 2 + encryptionV1SaltLen + encryptionNonceLen + encryptionMacLen
	EncryptionV2Overhead = 2 + encryptionV2EncryptedBlobLen + encryptionNonceLen + encryptionMacLen
)

var (
	encryptionVer  byte
	encryptionBlob []byte
	encryptionKey  []byte
)

func IsEncryptedData(data []byte) bool {
	return (len(data) > EncryptionV1Overhead || len(data) > EncryptionV2Overhead) &&
		data[0] == encryptionMarkerByte0 &&
		(data[1] == encryptionV1MarkerByte1 || data[1] == encryptionV2MarkerByte1)
}

// for password based key the blob is salt
// for AWS KMS the blob is encrypted data key
// if no blob is supplied then a new key is requested
// if ver is supplied then it must match envionment setup
func encryptionKeyInit(ver byte, blob []byte) (byte, []byte, []byte, error) {
	if ver == encryptionV1MarkerByte1 && config.CryptoPassword == "" {
		return 0, nil, nil,
			fmt.Errorf("Set HUB_CRYPTO_PASSWORD='random password'")
	}
	if ver == encryptionV2MarkerByte1 && config.CryptoAwsKmsKeyArn == "" {
		return 0, nil, nil,
			fmt.Errorf("Set HUB_CRYPTO_AWS_KMS_KEY_ARN='arn:aws:kms:...'")
	}
	if config.CryptoPassword != "" && (ver == 0 || ver == encryptionV1MarkerByte1) {
		salt := blob
		if len(salt) == 0 {
			salt = make([]byte, encryptionV1SaltLen)
			_, err := rand.Read(salt)
			if err != nil {
				return 0, nil, nil, err
			}
		}
		key := pbkdf2.Key([]byte(config.CryptoPassword), salt, 4096, 32, sha1.New)
		return encryptionV1MarkerByte1, salt, key, nil
	}
	if config.CryptoAwsKmsKeyArn != "" && (ver == 0 || ver == encryptionV2MarkerByte1) {
		clearKey, encryptedKey, err := aws.KmsKey(config.CryptoAwsKmsKeyArn, blob)
		if err != nil {
			return 0, nil, nil, err
		}
		return encryptionV2MarkerByte1, encryptedKey, clearKey, nil
	}
	return 0, nil, nil,
		fmt.Errorf("Set HUB_CRYPTO_PASSWORD='random password' or HUB_CRYPTO_AWS_KMS_KEY_ARN='arn:aws:kms:...'")
}

func maybeEncryptionKeyInit() (byte, []byte, []byte, error) {
	var err error
	if len(encryptionKey) == 0 {
		encryptionVer, encryptionBlob, encryptionKey, err = encryptionKeyInit(0, nil)
	}
	return encryptionVer, encryptionBlob, encryptionKey, err
}

func Encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	ver, blob, key, err := maybeEncryptionKeyInit()
	if err != nil {
		return nil, err
	}
	if ver == encryptionV2MarkerByte1 && len(blob) != encryptionV2EncryptedBlobLen {
		util.WarnOnce("AWS KMS `CiphertextBlob` size %d doesn't match built-in size %d",
			len(blob), encryptionV2EncryptedBlobLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if nonceSize != encryptionNonceLen {
		util.WarnOnce("Cipher `nonce` size %d doesn't match built-in size %d", nonceSize, encryptionNonceLen)
	}
	nonce := make([]byte, nonceSize)
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, data, blob)
	buf := bytes.NewBuffer(make([]byte, 0, 2+len(blob)+len(nonce)+len(ciphertext)))
	buf.WriteByte(encryptionMarkerByte0)
	buf.WriteByte(ver)
	buf.Write(blob)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

func Decrypt(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return encrypted, nil
	}
	if !IsEncryptedData(encrypted) {
		return nil, errors.New("Bad ciphertext marker")
	}

	overhead := EncryptionV1Overhead
	blobLen := encryptionV1SaltLen
	ver := encrypted[1]
	if ver == encryptionV2MarkerByte1 {
		overhead = EncryptionV2Overhead
		blobLen = encryptionV2EncryptedBlobLen
	}
	if len(encrypted) < overhead+aes.BlockSize {
		return nil, errors.New("Insufficient ciphertext length")
	}

	encrypted = encrypted[2:]
	blob := encrypted[:blobLen]
	rest := encrypted[blobLen:]

	_, _, key, err := encryptionKeyInit(ver, blob)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if nonceSize != encryptionNonceLen {
		util.WarnOnce("Cipher `nonce` size %d doesn't match built-in size %d", nonceSize, encryptionNonceLen)
	}
	nonce := rest[:nonceSize]
	ciphertext := rest[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, blob)
}
