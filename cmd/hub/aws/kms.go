// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	awsaws "github.com/aws/aws-sdk-go/aws"
	awskms "github.com/aws/aws-sdk-go/service/kms"
)

func KmsKey(arn string, blob []byte) ([]byte, []byte, error) {
	kms, err := awsKms(arnRegion(arn))
	if err != nil {
		return nil, nil, err
	}
	// new data key for encryption
	if len(blob) == 0 {
		resp, err := kms.GenerateDataKey(
			&awskms.GenerateDataKeyInput{
				KeyId:   &arn,
				KeySpec: awsaws.String("AES_256"),
			})
		if err != nil {
			return nil, nil, err
		}
		return resp.Plaintext, resp.CiphertextBlob, nil
	}
	// decrypt data key for decryption
	// TODO we may allow key ARN to be unset and retrieve it from resp.KeyId
	resp, err := kms.Decrypt(
		&awskms.DecryptInput{
			CiphertextBlob:      blob,
			EncryptionAlgorithm: awsaws.String("SYMMETRIC_DEFAULT"),
			KeyId:               &arn,
		})
	if err != nil {
		return nil, nil, err
	}
	return resp.Plaintext, blob, nil
}

func awsKms(region string) (*awskms.KMS, error) {
	session, err := Session(region, "KMS")
	if err != nil {
		return nil, err
	}
	return awskms.New(session), nil
}
