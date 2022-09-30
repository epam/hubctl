// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/epam/hubctl/cmd/hub/config"
)

var (
	defaultGcsClient *storage.Client
	gcsBuckets       = make(map[string]*storage.BucketHandle)
	gcsTimeout       = time.Duration(30 * time.Second)
)

func gcsClient() (*storage.Client, error) {
	if defaultGcsClient != nil {
		return defaultGcsClient, nil
	}
	ctx := context.Background()
	var err error
	opts := []option.ClientOption{option.WithScopes(storage.ScopeReadWrite)}
	if config.GcpCredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.GcpCredentialsFile))
	}
	defaultGcsClient, err = storage.NewClient(ctx, opts...)
	return defaultGcsClient, err
}

func gcsBucket(name string) (*storage.BucketHandle, error) {
	if bucket, exist := gcsBuckets[name]; exist {
		return bucket, nil
	}
	client, err := gcsClient()
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(name)
	gcsBuckets[name] = bucket
	return bucket, nil
}

func noRoot(path string) string {
	return strings.TrimLeft(path, "/")
}

func StatGCS(path string) (int64, time.Time, error) {
	location, err := url.Parse(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	bucket, err := gcsBucket(location.Host)
	if err != nil {
		return 0, time.Time{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcsTimeout)
	defer cancel()
	attrs, err := bucket.Object(noRoot(location.Path)).Attrs(ctx)
	if err != nil {
		if IsNotFound(err) {
			return 0, time.Time{}, os.ErrNotExist
		}
		return 0, time.Time{}, err
	}
	return attrs.Size, attrs.Updated, nil
}

func ReadGCS(path string) ([]byte, error) {
	location, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	bucket, err := gcsBucket(location.Host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcsTimeout)
	defer cancel()
	reader, err := bucket.Object(noRoot(location.Path)).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read GCS object `%s`: %v", path, err)
	}
	return data, nil
}

func WriteGCS(path string, body []byte) error {
	location, err := url.Parse(path)
	if err != nil {
		return err
	}
	bucket, err := gcsBucket(location.Host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcsTimeout)
	defer cancel()
	writer := bucket.Object(noRoot(location.Path)).NewWriter(ctx)
	defer writer.Close()
	written, err := writer.Write(body)
	if err != nil || written != len(body) {
		return fmt.Errorf("Failed to write GCS object `%s` (wrote %d of %d bytes): %v",
			path, written, len(body), err)
	}
	return nil
}
