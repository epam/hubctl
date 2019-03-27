package storage

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"hub/aws"
	"hub/config"
	"hub/gcp"
	"hub/util"
)

var storageSchemes = []string{"s3", "gs"}

func checkPath(path, kind string) (*File, error) {
	if strings.Contains(path, ",") {
		util.Warn("Did you split `%s` on ',' (comma)?", path)
	}
	if strings.Contains(path, "://") {
		remote, err := url.Parse(path)
		if err != nil {
			err = fmt.Errorf("Unable to parse `%s` %s file path as URL: %v", path, kind, err)
		} else if !util.Contains(storageSchemes, remote.Scheme) {
			err = fmt.Errorf("%s file `%s` scheme `%s` not supported. Supported schemes: %v",
				strings.Title(kind), path, remote.Scheme, storageSchemes)
		}
		if err != nil {
			return nil, err
		}
		return &File{Kind: remote.Scheme, Path: path}, nil
	} else {
		return &File{Kind: "fs", Path: path}, nil
	}
}

func Check(paths []string, kind string) (*Files, []error) {
	if len(paths) == 0 {
		return nil, nil
	}

	errs := make([]error, 0)

	files := make([]File, 0, len(paths))
	for _, path := range paths {
		file, err := checkPath(path, kind)
		if err != nil {
			errs = append(errs, err)
		} else {
			files = append(files, *file)
		}
	}

	filesChecked := make([]File, 0, len(files))
	for _, file := range files {
		lockPath := fmt.Sprintf("%s.lock", file.Path)
		switch file.Kind {
		case "fs":
			info, err := os.Stat(file.Path)
			_, errLock := os.Stat(lockPath)
			if err != nil {
				if util.NoSuchFile(err) {
					file.Exist = false
					file.Locked = errLock == nil
					filesChecked = append(filesChecked, file)
				} else {
					util.Warn("Unable to stat `%s` %s file: %v", file.Path, kind, err)
				}
			} else {
				file.Exist = true
				file.ModTime = info.ModTime()
				file.Size = info.Size()
				file.Locked = errLock == nil
				filesChecked = append(filesChecked, file)
			}

		case "s3":
			if config.Debug {
				log.Printf("Checking `%s` %s file...", file.Path, kind)
			}
			size, modTime, err := aws.StatS3(file.Path)
			_, _, errLock := aws.StatS3(lockPath)
			if err != nil {
				if err == os.ErrNotExist {
					file.Exist = false
					file.Locked = errLock == nil
					filesChecked = append(filesChecked, file)
				} else {
					util.Warn("Unable to check `%s` %s file: %v", file.Path, kind, err)
				}
			} else {
				file.Exist = true
				file.ModTime = modTime
				file.Size = size
				file.Locked = errLock == nil
				filesChecked = append(filesChecked, file)
			}

		case "gs":
			if config.Debug {
				log.Printf("Checking `%s` %s file...", file.Path, kind)
			}
			size, modTime, err := gcp.StatGCS(file.Path)
			_, _, errLock := gcp.StatGCS(lockPath)
			if err != nil {
				if err == os.ErrNotExist {
					file.Exist = false
					file.Locked = errLock == nil
					filesChecked = append(filesChecked, file)
				} else {
					util.Warn("Unable to check `%s` %s file: %v", file.Path, kind, err)
				}
			} else {
				file.Exist = true
				file.ModTime = modTime
				file.Size = size
				file.Locked = errLock == nil
				filesChecked = append(filesChecked, file)
			}
		}
	}

	if len(filesChecked) == 0 {
		return nil, append(errs, fmt.Errorf("No usable %s file found", kind))
	}

	if len(filesChecked) > 0 && (config.Trace || (config.Debug && kind != "manifest")) {
		printFiles(filesChecked, kind)
	}

	if len(errs) == 0 {
		errs = nil
	}

	return &Files{Kind: kind, Files: filesChecked}, errs
}

func EnsureNoLockFiles(files *Files) {
	locked := make([]string, 0)
	for _, file := range files.Files {
		if file.Locked {
			locked = append(locked, fmt.Sprintf("%s.lock", file.Path))
		}
	}
	if len(locked) > 0 {
		log.Fatalf("Lock %s %s present - delete to proceed",
			util.Plural(len(locked), "file"), strings.Join(locked, ", "))
	}
}

func chooseFile(files *Files) (*File, error) {
	delta := time.Duration(-10) * time.Second

	filesExist := make([]File, 0, len(files.Files))
	for _, file := range files.Files {
		if file.Exist {
			filesExist = append(filesExist, file)
		}
	}

	if len(filesExist) == 0 {
		return nil, os.ErrNotExist
	}
	if len(filesExist) == 1 {
		return &filesExist[0], nil
	}

	modTime := filesExist[0].ModTime
	for _, file := range filesExist {
		if file.ModTime.After(modTime) {
			modTime = file.ModTime
		}
	}
	modTime = modTime.Add(delta)
	candidates := make([]File, 0, len(filesExist))
	for _, file := range filesExist {
		if file.ModTime.After(modTime) {
			candidates = append(candidates, file)
		}
	}

	if len(candidates) == 1 {
		return &candidates[0], nil
	}

	largest := candidates[0]
	for _, file := range candidates {
		if file.Size > largest.Size {
			largest = file
		}
	}
	if largest.Kind == "fs" {
		return &largest, nil
	}
	for _, file := range candidates {
		if file.Kind == "fs" &&
			(file.Size == largest.Size || (file.Size+util.EncryptionOverhead == largest.Size)) {
			return &file, nil
		}
	}

	return &largest, nil
}

func readFile(file *File) ([]byte, error) {
	var data []byte
	var err error

	switch file.Kind {
	case "fs":
		data, err = ioutil.ReadFile(file.Path)

	case "s3":
		data, err = aws.ReadS3(file.Path)

	case "gs":
		data, err = gcp.ReadGCS(file.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("Unable to read `%s`: %v", file.Path, err)
	}
	return data, nil
}

func chooseAndReadFile(files *Files) ([]byte, string, error) {
	file, err := chooseFile(files)
	if err != nil {
		return nil, "", err
	}
	data, err := readFile(file)
	return data, file.Path, err
}

func Read(files *Files) ([]byte, string, error) {
	data, path, err := chooseAndReadFile(files)
	if err != nil {
		return nil, "", err
	}

	if util.IsEncryptedData(data) {
		data, err = util.Decrypt(data)
		if err != nil {
			return nil, "", fmt.Errorf("Unable to decrypt `%s`: %v", path, err)
		}
	}
	if util.IsGzipData(data) {
		data, err = util.Gunzip(data)
		if err != nil {
			return nil, "", fmt.Errorf("Unable to gunzip `%s`: %v", path, err)
		}
	}
	if config.Verbose {
		log.Printf("Read `%s` %s file", path, files.Kind)
	}

	return data, path, nil
}

func CheckAndRead(paths []string, kind string) ([]byte, string, error) {
	files, errs := Check(paths, kind)
	if len(errs) > 0 {
		return nil, "", fmt.Errorf("Unable to check %s files: %s", kind, util.Errors2(errs...))
	}
	return Read(files)
}
