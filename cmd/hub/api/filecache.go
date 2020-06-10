package api

import (
	"errors"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/config"
)

type AccessTokenBox struct {
	ApiBaseUrl     string
	LoginTokenHash uint64
	AccessToken    string
	RefreshToken   string
}

type FileCache struct {
	AccessTokens []AccessTokenBox
}

var crc64Table = crc64.MakeTable(crc64.ECMA)
var errorNoImpl = errors.New("Not implemented")

func hashLoginToken(loginToken string) uint64 {
	return crc64.Checksum([]byte(loginToken), crc64Table)
}

// TODO use golang.org/x/sys/unix FcntlFlock()
func readCache(flag int) (*os.File, *FileCache, error) {
	if config.CacheFile == "" {
		return nil, nil, errors.New("No cache file set, try --cache")
	}
	if config.Trace {
		log.Printf("Opening `%s`", config.CacheFile)
	}
	file, err := os.OpenFile(config.CacheFile, flag, 0640)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	yamlBytes, err := ioutil.ReadAll(file)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if len(yamlBytes) == 0 {
		return file, nil, nil
	}
	var cache FileCache
	err = yaml.Unmarshal(yamlBytes, &cache)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return file, &cache, nil
}

func loadAccessToken(apiBaseUrl, loginToken string) (*SigninResponse, error) {
	file, cache, err := readCache(os.O_RDONLY)
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
	file, cache, err := readCache(os.O_RDWR | os.O_CREATE)
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
		cache = &FileCache{AccessTokens: make([]AccessTokenBox, 0, 1)}
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
		box := AccessTokenBox{ApiBaseUrl: apiBaseUrl, LoginTokenHash: ltHash, AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}
		cache.AccessTokens = append(cache.AccessTokens, box)
	}

	yamlBytes, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	wrote, err := file.Write(yamlBytes)
	if err != nil {
		return err
	}
	if wrote != len(yamlBytes) {
		return fmt.Errorf("Wrote %d out of %d bytes", wrote, len(yamlBytes))
	}
	return nil
}
