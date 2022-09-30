// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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

	"github.com/epam/hubctl/cmd/hub/aws"
	"github.com/epam/hubctl/cmd/hub/azure"
	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/gcp"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	aes256KeySize = 32

	// V1 is pbkdf2 password derived key
	// V2 is AWS KMS
	// V3 is Azure KeyVault
	// V4 is GCP KMS
	encryptionMarkerByte0        = '\x26'
	encryptionV1MarkerByte1      = '\x01'
	encryptionV2MarkerByte1      = '\x02'
	encryptionV3MarkerByte1      = '\x03'
	encryptionV4MarkerByte1      = '\x04'
	encryptionV1SaltLen          = 8
	encryptionNonceLen           = 12
	encryptionV2EncryptedBlobLen = 184 // encrypted AES256 key and 152 bytes of fixed-size AWS KMS meta
	encryptionV3EncryptedBlobLen = 256 // RSA-OAEP-256
	encryptionV4EncryptedBlobLen = 113 // encrypted AES256 key and 81 bytes of fixed-size GCP KMS meta
	encryptionMacLen             = 16

	EncryptionV1Overhead = 2 + encryptionV1SaltLen + encryptionNonceLen + encryptionMacLen
	EncryptionV2Overhead = 2 + encryptionV2EncryptedBlobLen + encryptionNonceLen + encryptionMacLen
	EncryptionV3Overhead = 2 + encryptionV3EncryptedBlobLen + encryptionNonceLen + encryptionMacLen
	EncryptionV4Overhead = 2 + encryptionV4EncryptedBlobLen + encryptionNonceLen + encryptionMacLen

	helpPassword      = "HUB_CRYPTO_PASSWORD='random password'"
	helpAwsKms        = "HUB_CRYPTO_AWS_KMS_KEY_ARN='arn:aws:kms:...'"
	helpAzukeKeyvault = "HUB_CRYPTO_AZURE_KEYVAULT_KEY_ID='https://*.vault.azure.net/keys/...'"
	helpGcpKms        = "HUB_CRYPTO_GCP_KMS_KEY_NAME='projects/*/locations/*/keyRings/my-key-ring/cryptoKeys/my-key'"
)

var (
	encryptionVer  byte
	encryptionBlob []byte
	encryptionKey  []byte
)

func IsEncryptedData(data []byte) bool {
	return (len(data) > EncryptionV1Overhead || len(data) > EncryptionV2Overhead || len(data) > EncryptionV3Overhead || len(data) > EncryptionV4Overhead) &&
		data[0] == encryptionMarkerByte0 &&
		(data[1] == encryptionV1MarkerByte1 || data[1] == encryptionV2MarkerByte1 || data[1] == encryptionV3MarkerByte1 || data[1] == encryptionV4MarkerByte1)
}

// for password based key the blob is salt
// for AWS KMS, Azure Key Vault, GCP KMS the blob is encrypted data key
// if no blob is supplied then a new key is requested
// if ver is supplied then it must match envionment setup
func encryptionKeyInit(ver byte, blob []byte) (byte, []byte, []byte, error) {
	if ver == encryptionV1MarkerByte1 && config.CryptoPassword == "" {
		return 0, nil, nil,
			fmt.Errorf("Set %s", helpPassword)
	}
	if ver == encryptionV2MarkerByte1 && config.CryptoAwsKmsKeyArn == "" {
		return 0, nil, nil,
			fmt.Errorf("Set %s", helpAwsKms)
	}
	if ver == encryptionV3MarkerByte1 && config.CryptoAzureKeyVaultKeyId == "" {
		return 0, nil, nil,
			fmt.Errorf("Set %s", helpAzukeKeyvault)
	}
	if ver == encryptionV4MarkerByte1 && config.CryptoGcpKmsKeyName == "" {
		return 0, nil, nil,
			fmt.Errorf("Set %s", helpGcpKms)
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
		key := pbkdf2.Key([]byte(config.CryptoPassword), salt, 4096, aes256KeySize, sha1.New)
		return encryptionV1MarkerByte1, salt, key, nil
	}
	if config.CryptoAwsKmsKeyArn != "" && (ver == 0 || ver == encryptionV2MarkerByte1) {
		clearKey, encryptedKey, err := aws.KmsKey(config.CryptoAwsKmsKeyArn, blob)
		if err != nil {
			return 0, nil, nil, err
		}
		return encryptionV2MarkerByte1, encryptedKey, clearKey, nil
	}
	if config.CryptoAzureKeyVaultKeyId != "" && (ver == 0 || ver == encryptionV3MarkerByte1) {
		clearKey, encryptedKey, err := azure.KeyvaultKey(config.CryptoAzureKeyVaultKeyId, blob)
		if err != nil {
			return 0, nil, nil, err
		}
		return encryptionV3MarkerByte1, encryptedKey, clearKey, nil
	}
	if config.CryptoGcpKmsKeyName != "" && (ver == 0 || ver == encryptionV4MarkerByte1) {
		clearKey, encryptedKey, err := gcp.KmsKey(config.CryptoGcpKmsKeyName, blob)
		if err != nil {
			return 0, nil, nil, err
		}
		return encryptionV4MarkerByte1, encryptedKey, clearKey, nil
	}
	return 0, nil, nil,
		fmt.Errorf("Set %s or %s or %s", helpPassword, helpAwsKms, helpAzukeKeyvault)
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
	if ver == encryptionV3MarkerByte1 && len(blob) != encryptionV3EncryptedBlobLen {
		util.WarnOnce("Azure Key Vault encrypted key size %d doesn't match built-in size %d",
			len(blob), encryptionV3EncryptedBlobLen)
	}
	if ver == encryptionV4MarkerByte1 && len(blob) != encryptionV4EncryptedBlobLen {
		util.WarnOnce("GCP KMS encrypted key size %d doesn't match built-in size %d",
			len(blob), encryptionV4EncryptedBlobLen)
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
	} else if ver == encryptionV3MarkerByte1 {
		overhead = EncryptionV3Overhead
		blobLen = encryptionV3EncryptedBlobLen
	} else if ver == encryptionV4MarkerByte1 {
		overhead = EncryptionV4Overhead
		blobLen = encryptionV4EncryptedBlobLen
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
