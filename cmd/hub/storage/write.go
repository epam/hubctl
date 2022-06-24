// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package storage

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/agilestacks/hub/cmd/hub/aws"
	"github.com/agilestacks/hub/cmd/hub/azure"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/crypto"
	"github.com/agilestacks/hub/cmd/hub/gcp"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func Write(data []byte, files *Files) (bool, []error) {
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
			return false, []error{fmt.Errorf("Unable to gzip: %v", err)}
		}
		if config.Compressed {
			data = compressedData
		}
	}

	encryptedData := data
	if encrypt {
		var err error
		encryptedData, err = crypto.Encrypt(compressedData)
		if err != nil {
			return false, []error{fmt.Errorf("Unable to encrypt: %v", err)}
		}
	}

	var errs []error
	written := false
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

		if nErrs == 0 {
			if config.Verbose {
				log.Printf("Wrote %s `%s`", files.Kind, file.Path)
			}
			written = true
		}
	}

	return written, errs
}
