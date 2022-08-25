// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package api

import (
	"errors"
	"hash/crc64"
	"os"

	"github.com/agilestacks/hub/cmd/hub/filecache"
)

var crc64Table = crc64.MakeTable(crc64.ECMA)

func hashLoginToken(loginToken string) uint64 {
	return crc64.Checksum([]byte(loginToken), crc64Table)
}

func loadAccessToken(apiBaseUrl, loginToken string) (*SigninResponse, error) {
	file, cache, err := filecache.ReadCache(os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	if file != nil {
		file.Close()
	}
	if cache != nil {
		ltHash := hashLoginToken(loginToken)
		for _, box := range cache.AccessTokens {
			if apiBaseUrl == box.ApiBaseUrl && ltHash == box.LoginTokenHash {
				return &SigninResponse{AccessToken: box.AccessToken, RefreshToken: box.RefreshToken}, nil
			}
		}
	}
	return nil, nil
}

func storeAccessToken(apiBaseUrl, loginToken string, tokens *SigninResponse) error {
	file, cache, err := filecache.ReadCache(os.O_RDWR | os.O_CREATE)
	if err != nil {
		return err
	}
	if file == nil {
		return errors.New("No cache file created")
	}
	defer file.Close()
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	if cache == nil {
		cache = &filecache.FileCache{AccessTokens: make([]filecache.AccessTokenBox, 0, 1)}
	}

	ltHash := hashLoginToken(loginToken)
	found := false
	for i, b := range cache.AccessTokens {
		if apiBaseUrl == b.ApiBaseUrl && ltHash == b.LoginTokenHash {
			box := &cache.AccessTokens[i]
			box.AccessToken = tokens.AccessToken
			box.RefreshToken = tokens.RefreshToken
			found = true
			break
		}
	}
	if !found {
		box := filecache.AccessTokenBox{ApiBaseUrl: apiBaseUrl, LoginTokenHash: ltHash, AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}
		cache.AccessTokens = append(cache.AccessTokens, box)
	}

	return filecache.WriteCache(file, cache)
}
