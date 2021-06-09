/*
Copyright 2018, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

// S3Config holds all the configuration params passed
type S3Config struct {
	region                   string
	s3URL                    string
	publicURL                string
	kmsKeyID                 string
	s3ForcePathStyleVal      string
	signatureVersion         string
	credentialProfile        string
	credentialsFile          string
	serverSideEncryption     string
	insecureSkipTLSVerifyVal string
	bucket                   string
	caCert                   string
	session                  *session.Session
	publicSession            *session.Session
}

// NewS3Config returns an instance of the config loaded from by the plugin framework
func NewS3Config(cfg map[string]string) *S3Config {
	return &S3Config{
		region:                   cfg[regionKey],
		s3URL:                    cfg[s3URLKey],
		publicURL:                cfg[publicURLKey],
		kmsKeyID:                 cfg[kmsKeyIDKey],
		s3ForcePathStyleVal:      cfg[s3ForcePathStyleKey],
		signatureVersion:         cfg[signatureVersionKey],
		credentialProfile:        cfg[credentialProfileKey],
		credentialsFile:          cfg[credentialsFileKey],
		serverSideEncryption:     cfg[serverSideEncryptionKey],
		insecureSkipTLSVerifyVal: cfg[insecureSkipTLSVerifyKey],
		bucket:                   cfg[bucketKey],
		caCert:                   cfg[caCertKey],
	}
}

// Init initializes the S3 config parameters which needs to be fetched/prepared based on the passed options
func (s *S3Config) Init() error {
	// AWS (not an alternate S3-compatible API) and region not
	// explicitly specified: determine the bucket's region
	var (
		s3ForcePathStyle      bool
		insecureSkipTLSVerify bool
		err                   error
	)

	if s.s3ForcePathStyleVal != "" {
		if s3ForcePathStyle, err = strconv.ParseBool(s.s3ForcePathStyleVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", s3ForcePathStyleKey)
		}
	}

	// prepares the AWS Sessions with the session-config having only known parameters.
	// Any unknown or derived params will be added further to the initialization
	if err = s.initAWSSessions(s3ForcePathStyle); err != nil {
		return errors.Wrap(err, "could not initialize AWS session")
	}

	// add region to the session config
	if s.s3URL == "" && s.region == "" {
		var region string
		var fErr error

		for _, partition := range endpoints.DefaultPartitions() {
			for regionHint := range partition.Regions() {
				var err error
				region, err = s3manager.GetBucketRegion(context.Background(), s.session, s.bucket, regionHint)
				if err != nil {
					fErr = errors.Wrap(fErr, err.Error())
				}
				// we only need to try a single region hint per partition, so break after the first
				break
			}

			if region != "" {
				s.region = region
				break
			}
		}

		if region == "" {
			return errors.Wrap(fErr, "unable to determine bucket's region")
		}

		s.session.Config = s.session.Config.WithRegion(s.region)
	}

	// add insecure flag to the http transport config in http client and add it to the session config
	if s.insecureSkipTLSVerifyVal != "" {
		if insecureSkipTLSVerify, err = strconv.ParseBool(s.insecureSkipTLSVerifyVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", insecureSkipTLSVerifyKey)
		}
	}

	if insecureSkipTLSVerify {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		s.session.Config.HTTPClient = &http.Client{
			// Copied from net/http
			Transport: &http.Transport{
				Proxy:                 defaultTransport.Proxy,
				DialContext:           defaultTransport.DialContext,
				MaxIdleConns:          defaultTransport.MaxIdleConns,
				IdleConnTimeout:       defaultTransport.IdleConnTimeout,
				TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
				ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
				// Set insecureSkipVerify true
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return nil
}

func (s *S3Config) initAWSSessions(s3ForcePathStyle bool) error {
	cfg, err := newAWSConfig(s.s3URL, s.region, s3ForcePathStyle)
	if err != nil {
		return err
	}

	sessionOptions, err := newSessionOptions(*cfg, s.credentialProfile, s.caCert, s.credentialsFile)
	if err != nil {
		return err
	}

	s.session, err = getSession(sessionOptions)
	if err != nil {
		return errors.WithStack(err)
	}

	// init public session
	if s.publicURL != "" {
		pcfg, err := newAWSConfig(s.publicURL, s.region, s3ForcePathStyle)
		if err != nil {
			return err
		}

		publicSessionOptions, err := newSessionOptions(*pcfg, s.credentialProfile, s.caCert, s.credentialsFile)
		if err != nil {
			return err
		}

		s.publicSession, err = getSession(publicSessionOptions)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// IsValidS3URLScheme returns true if the scheme is http:// or https://
// and the url parses correctly, otherwise, return false
func IsValidS3URLScheme(s3URL string) bool {
	u, err := url.Parse(s3URL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return true
}

// takes AWS session options to create a new session
func getSession(options session.Options) (*session.Session, error) {
	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStack(err)
	}
	return sess, nil
}
