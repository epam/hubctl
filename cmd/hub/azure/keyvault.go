// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package azure

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	keyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
)

const aes256KeySize = 32

var (
	keyvaultTimeout = time.Duration(10 * time.Second)
	keyvaultKeyRe   = regexp.MustCompile("^(https://[^/]+)/keys/([^/]+)/([^/]+)$")
)

func KeyvaultKey(id string, blob []byte) ([]byte, []byte, error) {
	vault, keyName, keyVersion, err := keySplit(id)
	if err != nil {
		return nil, nil, err
	}

	auth, err := authorizer(keyvaultResource)
	if err != nil {
		return nil, nil, err
	}
	kv := keyvault.New()
	kv.Authorizer = auth
	ctx, cancel := context.WithTimeout(context.Background(), keyvaultTimeout)
	defer cancel()

	// decrypt data key
	data := blob
	op := kv.Decrypt

	generate := len(blob) == 0
	// new data key for encryption
	var key []byte
	if generate {
		key = make([]byte, aes256KeySize)
		_, err := rand.Read(key)
		if err != nil {
			return nil, nil, err
		}
		data = key
		op = kv.Encrypt
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	resp, err := op(ctx, vault, keyName, keyVersion,
		keyvault.KeyOperationsParameters{Value: &encoded, Algorithm: keyvault.RSAOAEP256})
	if err != nil {
		return nil, nil, err
	}
	decoded, err := base64.RawURLEncoding.DecodeString(*resp.Result)
	if err != nil {
		return nil, nil, err
	}

	if generate {
		return key, decoded, nil
	}
	return decoded, blob, nil
}

func keySplit(id string) (string, string, string, error) {
	p := keyvaultKeyRe.FindStringSubmatch(id)
	if len(p) != 4 {
		return "", "", "", fmt.Errorf("Unable to parse Azure Key Vault key id `%s`; the correct format is https://<vault name>.vault.azure.net/keys/<key name>/<key version>", id)
	}
	return p[1], p[2], p[3], nil
}
