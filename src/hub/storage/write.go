package storage

import (
	"errors"
	"fmt"
	"log"
	"os"

	"hub/aws"
	"hub/azure"
	"hub/config"
	"hub/gcp"
	"hub/util"
)

func Write(data []byte, files *Files) []error {
	// write remote files encrypted
	encrypt := false
	if config.Encrypted {
		for _, file := range files.Files {
			if util.Contains(remoteStorageSchemes, file.Kind) {
				encrypt = true
				break
			}
		}
	}

	var compressedData []byte
	if config.Compressed || encrypt {
		var err error
		compressedData, err = util.Gzip(data)
		if err != nil {
			return []error{fmt.Errorf("Unable to gzip: %v", err)}
		}
		if config.Compressed {
			data = compressedData
		}
	}

	encryptedData := data
	if encrypt {
		var err error
		encryptedData, err = util.Encrypt(compressedData)
		if err != nil {
			return []error{fmt.Errorf("Unable to encrypt: %v", err)}
		}
	}

	errs := make([]error, 0)

	for _, file := range files.Files {
		nErrs := len(errs)
		switch file.Kind {
		case "s3":
			err := aws.WriteS3(file.Path, encryptedData)
			if err != nil {
				msg := fmt.Sprintf("Unable to write `%s` %s file: %v", file.Path, files.Kind, err)
				if aws.IsSlowDown(err) && (len(files.Files) > 1 || config.Force) {
					util.Warn("%s", msg)
				} else {
					errs = append(errs, errors.New(msg))
				}
			}

		case "gs":
			err := gcp.WriteGCS(file.Path, encryptedData)
			if err != nil {
				msg := fmt.Sprintf("Unable to write `%s` %s file: %v", file.Path, files.Kind, err)
				errs = append(errs, errors.New(msg))
			}

		case "az":
			err := azure.WriteStorageBlob(file.Path, encryptedData)
			if err != nil {
				msg := fmt.Sprintf("Unable to write `%s` %s file: %v", file.Path, files.Kind, err)
				errs = append(errs, errors.New(msg))
			}

		case "fs":
			out, err := os.Create(file.Path)
			if err != nil {
				err = fmt.Errorf("Unable to open `%s` %s file for write: %v", file.Path, files.Kind, err)
				errs = append(errs, err)
				continue
			}
			wrote, err := out.Write(data)
			err2 := out.Close()
			if err != nil || wrote != len(data) || err2 != nil {
				if err == nil && err2 != nil {
					err = err2
				}
				err = fmt.Errorf("Unable to write `%s` %s file (wrote %d out of %d bytes): %s",
					file.Path, files.Kind, wrote, len(data), util.Errors2(err))
				errs = append(errs, err)
			}
		}

		if config.Verbose && nErrs == len(errs) {
			log.Printf("Wrote %s `%s`", files.Kind, file.Path)
		}
	}

	if len(errs) == 0 {
		errs = nil
	}

	return errs
}
