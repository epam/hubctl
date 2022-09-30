// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	awsaws "github.com/aws/aws-sdk-go/aws"
	awss3 "github.com/aws/aws-sdk-go/service/s3"

	"github.com/epam/hubctl/cmd/hub/config"
)

var (
	bucketRegion = make(map[string]string)
	regionS3     = make(map[string]*awss3.S3)
)

func awsBucketS3(bucket string) (*awss3.S3, error) {
	region, err := awsBucketRegion(bucket)
	if err != nil {
		return nil, err
	}
	return awsS3(region)
}

func awsBucketRegion(bucket string) (string, error) {
	if region, exist := bucketRegion[bucket]; exist {
		return region, nil
	}
	s3, err := awsS3(config.AwsRegion)
	if err != nil {
		return "", err
	}
	location, err := s3.GetBucketLocation(
		&awss3.GetBucketLocationInput{
			Bucket: &bucket,
		})
	if err != nil {
		return "", fmt.Errorf("Unable to determine AWS bucket `%s` region: %v", bucket, err)
	}
	region := "us-east-1"
	if location.LocationConstraint != nil && *location.LocationConstraint != "" {
		region = *location.LocationConstraint
	}
	if config.Debug {
		log.Printf("S3 bucket `%s` region is %s", bucket, region)
	}
	bucketRegion[bucket] = region
	return region, nil
}

func awsS3(region string) (*awss3.S3, error) {
	session, err := Session(region, "S3")
	if err != nil {
		return nil, err
	}
	s3 := awss3.New(session)
	regionS3[region] = s3
	return s3, nil
}

func StatS3(s3path string) (int64, time.Time, error) {
	location, err := url.Parse(s3path)
	if err != nil {
		return 0, time.Time{}, err
	}
	s3, err := awsBucketS3(location.Host)
	if err != nil {
		return 0, time.Time{}, err
	}
	head, err := s3.HeadObject(
		&awss3.HeadObjectInput{
			Bucket: &location.Host,
			Key:    &location.Path,
		})
	if err != nil {
		if IsNotFound(err) {
			return 0, time.Time{}, os.ErrNotExist
		}
		return 0, time.Time{}, fmt.Errorf("Failed to HEAD S3 object `%s`: %v\n\t%s", s3path, err, optionsHelp)
	}
	return *head.ContentLength, *head.LastModified, nil
}

func ReadS3(s3path string) ([]byte, error) {
	location, err := url.Parse(s3path)
	if err != nil {
		return nil, err
	}
	s3, err := awsBucketS3(location.Host)
	if err != nil {
		return nil, err
	}
	obj, err := s3.GetObject(
		&awss3.GetObjectInput{
			Bucket: &location.Host,
			Key:    &location.Path,
		})
	if err != nil {
		return nil, fmt.Errorf("Failed to GET S3 object `%s`: %v\n\t%s", s3path, err, optionsHelp)
	}
	data, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read S3 object `%s`: %v", s3path, err)
	}
	obj.Body.Close()
	return data, nil
}

func WriteS3(s3path string, body []byte) error {
	location, err := url.Parse(s3path)
	if err != nil {
		return err
	}
	s3, err := awsBucketS3(location.Host)
	if err != nil {
		return err
	}
	_, err = s3.PutObject(
		&awss3.PutObjectInput{
			Body:   awsaws.ReadSeekCloser(bytes.NewReader(body)),
			Bucket: &location.Host,
			Key:    &location.Path,
		})
	if err != nil {
		return fmt.Errorf("Failed to PUT S3 object `%s`: %v\n\t%s", s3path, err, optionsHelp)
	}
	return nil
}
