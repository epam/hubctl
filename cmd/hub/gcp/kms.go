package gcp

import (
	"context"
	"crypto/rand"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

const aes256KeySize = 32

var (
	kmsTimeout = time.Duration(10 * time.Second)
)

func KmsKey(name string, blob []byte) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kmsTimeout)
	defer cancel()
	kms, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer kms.Close()

	// new data key for encryption
	if len(blob) == 0 {
		key := make([]byte, aes256KeySize)
		_, err := rand.Read(key)
		if err != nil {
			return nil, nil, err
		}
		req := &kmspb.EncryptRequest{
			Name:      name,
			Plaintext: key,
		}
		result, err := kms.Encrypt(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		return key, result.Ciphertext, nil
	}
	// decrypt data key for decryption
	req := &kmspb.DecryptRequest{
		Name:       name,
		Ciphertext: blob,
	}
	result, err := kms.Decrypt(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return result.Plaintext, blob, nil
}
