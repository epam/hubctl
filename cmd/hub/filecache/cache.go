package filecache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/config"
)

// TODO use golang.org/x/sys/unix FcntlFlock()
func ReadCache(flag int) (*os.File, *FileCache, error) {
	if config.CacheFile == "" {
		return nil, nil, errors.New("No cache file set, try --cache")
	}
	if config.Trace {
		log.Printf("Opening `%s` mode %d", config.CacheFile, flag)
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

func WriteCache(file *os.File, cache *FileCache) error {
	if cache.Version == 0 {
		cache.Version = 1
	}
	yamlBytes, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	if config.Trace {
		log.Printf("Writing `%s`", config.CacheFile)
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
