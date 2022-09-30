// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"fmt"
	"log"
	"os"

	awsaws "github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	awsec2rolecreds "github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	awsec2metadata "github.com/aws/aws-sdk-go/aws/ec2metadata"
	awssession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/epam/hubctl/cmd/hub/config"
)

var (
	credentialsCache = make(map[string]*awscredentials.Credentials)
)

const optionsHelp = "Try --aws_profile and --aws_region command-line options"

func cachedCredentials(purpose string) *awscredentials.Credentials {
	creds, exist := credentialsCache[purpose]
	if !exist {
		creds = DefaultCredentials(purpose)
		credentialsCache[purpose] = creds
	}
	return creds
}

func ProfileCredentials(profile, purpose string) *awscredentials.Credentials {
	if config.Debug {
		printEC2Metadata := ""
		if config.AwsUseIamRoleCredentials {
			printEC2Metadata = " and EC2 metadata"
		}
		if purpose != "" {
			purpose = fmt.Sprintf(" to access %s", purpose)
		}
		log.Printf("Asking `%s` AWS profile%s for credentials%s",
			profile, printEC2Metadata, purpose)
	}
	shared := &awscredentials.SharedCredentialsProvider{}
	if profile != "" {
		shared.Profile = profile
	}
	env := &awscredentials.EnvProvider{}
	providers := []awscredentials.Provider{env, shared}
	if config.AwsPreferProfileCredentials {
		providers = []awscredentials.Provider{shared, env}
	}
	if config.AwsUseIamRoleCredentials {
		awsSession, _ := awssession.NewSession()
		providers = append(providers, &awsec2rolecreds.EC2RoleProvider{Client: awsec2metadata.New(awsSession)})
	}
	return awscredentials.NewCredentials(&awscredentials.ChainProvider{Providers: providers, VerboseErrors: config.Verbose})
}

func DefaultCredentials(purpose string) *awscredentials.Credentials {
	profile := "default"
	if envProfile := os.Getenv("AWS_PROFILE"); envProfile != "" {
		profile = envProfile
	}
	if config.AwsProfile != "" {
		profile = config.AwsProfile
	}
	return ProfileCredentials(profile, purpose)
}

func Session(region, purpose string) (*awssession.Session, error) {
	var session *awssession.Session
	var err error
	if config.AwsProfile != "" || !config.AwsUseIamRoleCredentials || config.AwsPreferProfileCredentials {
		session, err = SessionWithCredentials(region, purpose, cachedCredentials(purpose))
	} else {
		session, err = SessionWithSharedConfig(region)
	}
	if err != nil {
		if purpose != "" {
			purpose = fmt.Sprintf(" for %s", purpose)
		}
		return nil, fmt.Errorf("Error initializing AWS session%s: %v", purpose, err)
	}
	return session, nil
}

func SessionWithStaticCredentials(region, purpose, accessKey, secretKey, token string) (*awssession.Session, error) {
	return SessionWithCredentials(region, purpose, awscredentials.NewStaticCredentials(accessKey, secretKey, token))
}

func SessionWithCredentials(region, purpose string, credentials *awscredentials.Credentials) (*awssession.Session, error) {
	awsConfig := awsaws.NewConfig()
	if region != "" {
		awsConfig = awsConfig.WithRegion(region)
	}
	awsConfig = awsConfig.WithCredentials(credentials)
	return awssession.NewSession(awsConfig)
}

func SessionWithSharedConfig(region string) (*awssession.Session, error) {
	options := awssession.Options{SharedConfigState: awssession.SharedConfigEnable}
	if region != "" {
		options.Config = *awsaws.NewConfig().WithRegion(region)
	}
	return awssession.NewSessionWithOptions(options)
}
