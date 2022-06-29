// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package azure

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"

	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	blobClients       = make(map[string]*storage.BlobStorageClient)
	storageTimeoutSec = uint(10)
	storageTimeout    = time.Duration((10 + 1) * time.Second)
)

func storageClient(account string) (*storage.BlobStorageClient, error) {
	if client, exist := blobClients[account]; exist {
		return client, nil
	}
	_, env, err := settings()
	if err != nil {
		return nil, err
	}
	key, err := storageKey(account)
	if err != nil {
		return nil, err
	}
	storageClient, err := storage.NewClient(account, key, env.StorageEndpointSuffix, storage.DefaultAPIVersion, true)
	storageClient.HTTPClient = util.RobustHttpClient(storageTimeout, false)
	blobClient := storageClient.GetBlobService()
	blobClients[account] = &blobClient
	return &blobClient, nil
}

func splitPath(path string) (string, string, string, error) {
	location, err := url.Parse(path)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.SplitN(location.Path, "/", 3)
	if len(parts) != 3 {
		return "", "", "", errors.New("Bad path format")
	}
	return location.Host, parts[1], parts[2], nil
}

func StatStorageBlob(path string) (int64, time.Time, error) {
	account, container, name, err := splitPath(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	blobClient, err := storageClient(account)
	if err != nil {
		return 0, time.Time{}, err
	}
	containerRef := blobClient.GetContainerReference(container)
	blobRef := containerRef.GetBlobReference(name)
	err = blobRef.GetProperties(nil)
	if err != nil {
		if IsNotFound(err) {
			return 0, time.Time{}, os.ErrNotExist
		}
		return 0, time.Time{}, err
	}
	props := blobRef.Properties
	return props.ContentLength, time.Time(props.LastModified), nil
}

func ReadStorageBlob(path string) ([]byte, error) {
	account, container, name, err := splitPath(path)
	if err != nil {
		return nil, err
	}
	blobClient, err := storageClient(account)
	if err != nil {
		return nil, err
	}
	containerRef := blobClient.GetContainerReference(container)
	blobRef := containerRef.GetBlobReference(name)
	reader, err := blobRef.Get(&storage.GetBlobOptions{Timeout: storageTimeoutSec})
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read Azure storage blob `%s`: %v", path, err)
	}
	return data, nil
}

func WriteStorageBlob(path string, body []byte) error {
	account, container, name, err := splitPath(path)
	if err != nil {
		return err
	}
	blobClient, err := storageClient(account)
	if err != nil {
		return err
	}
	containerRef := blobClient.GetContainerReference(container)
	blobRef := containerRef.GetBlobReference(name)
	err = blobRef.PutAppendBlob(&storage.PutBlobOptions{Timeout: storageTimeoutSec})
	if err != nil {
		return err
	}
	err = blobRef.AppendBlock(body, &storage.AppendBlockOptions{Timeout: storageTimeoutSec})
	if err != nil {
		return fmt.Errorf("Failed to write Azure storage blob `%s`: %v", path, err)
	}
	return nil
}
