// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"encoding/base64"
	"fmt"

	awssession "github.com/aws/aws-sdk-go/aws/session"
	awseks "github.com/aws/aws-sdk-go/service/eks"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

var (
	eksSupportedRegions = []string{"us-east-1", "us-east-2", "us-west-2", "eu-west-1", "eu-central-1", "eu-north-1",
		"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2"}
)

func DescribeEKSCluster(region, name string) (string, []byte, error) {
	eks, err := awsEKS(config.AwsRegion)
	if err != nil {
		return "", nil, err
	}
	return describeEKSCluster(eks, name)
}

func DescribeEKSClusterWithStaticCredentials(region, name, accessKey, secretKey, token string) (string, []byte, error) {
	eks, err := awsEKSWithStaticCredentials(region, accessKey, secretKey, token)
	if err != nil {
		return "", nil, err
	}
	return describeEKSCluster(eks, name)
}

func describeEKSCluster(eks *awseks.EKS, name string) (string, []byte, error) {
	output, err := eks.DescribeCluster(&awseks.DescribeClusterInput{Name: &name})
	if err != nil {
		return "", nil, err
	}
	cluster := output.Cluster
	if cluster.Status == nil || *cluster.Status != awseks.ClusterStatusActive {
		status := "(nil)"
		if cluster.Status != nil {
			status = *cluster.Status
		}
		return "", nil, fmt.Errorf("Cluster `%s` status is `%s`; it must be `%s` to import",
			name, status, awseks.ClusterStatusActive)
	}
	endpoint := ""
	if cluster.Endpoint == nil {
		util.Warn("Empty cluster `%s` endpoint is returned by AWS API", name)
	} else {
		endpoint = *cluster.Endpoint
	}
	var cert []byte
	if cluster.CertificateAuthority == nil || cluster.CertificateAuthority.Data == nil || *cluster.CertificateAuthority.Data == "" {
		util.Warn("Empty cluster `%s` certificate authority is returned by AWS API", name)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(*cluster.CertificateAuthority.Data)
		if err != nil {
			return "", nil, fmt.Errorf("Unable to base64-decode cluster `%s` certificate authority: %v",
				name, err)
		}
		cert = decoded
	}
	return endpoint, cert, nil
}

func awsEKS(region string) (*awseks.EKS, error) {
	session, err := Session(region, "EKS")
	if err != nil {
		return nil, err
	}
	return awsEKSWithSession(region, session), nil
}

func awsEKSWithStaticCredentials(region, accessKey, secretKey, token string) (*awseks.EKS, error) {
	session, err := SessionWithStaticCredentials(region, "EKS", accessKey, secretKey, token)
	if err != nil {
		return nil, err
	}
	return awsEKSWithSession(region, session), nil
}

func awsEKSWithSession(region string, session *awssession.Session) *awseks.EKS {
	if region != "" && !util.Contains(eksSupportedRegions, region) {
		util.Warn("EKS might not be supported in `%s` region", region)
	}
	return awseks.New(session)
}
