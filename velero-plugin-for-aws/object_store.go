/*
Copyright the Velero contributors.

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
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

const (
	s3URLKey                     = "s3Url"
	publicURLKey                 = "publicUrl"
	kmsKeyIDKey                  = "kmsKeyId"
	customerKeyEncryptionFileKey = "customerKeyEncryptionFile"
	s3ForcePathStyleKey          = "s3ForcePathStyle"
	bucketKey                    = "bucket"
	signatureVersionKey          = "signatureVersion"
	credentialsFileKey           = "credentialsFile"
	credentialProfileKey         = "profile"
	serverSideEncryptionKey      = "serverSideEncryption"
	insecureSkipTLSVerifyKey     = "insecureSkipTLSVerify"
	caCertKey                    = "caCert"
	enableSharedConfigKey        = "enableSharedConfig"
)

type s3Interface interface {
	HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	ListObjectsV2Pages(input *s3.ListObjectsV2Input, fn func(*s3.ListObjectsV2Output, bool) bool) error
	DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	GetObjectRequest(input *s3.GetObjectInput) (req *request.Request, output *s3.GetObjectOutput)
}

type ObjectStore struct {
	log                  logrus.FieldLogger
	s3                   s3Interface
	preSignS3            s3Interface
	s3Uploader           *s3manager.Uploader
	kmsKeyID             string
	sseCustomerKey       string
	signatureVersion     string
	serverSideEncryption string
}

func newObjectStore(logger logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: logger}
}

func isValidSignatureVersion(signatureVersion string) bool {
	switch signatureVersion {
	case "1", "4":
		return true
	}
	return false
}

func (o *ObjectStore) Init(config map[string]string) error {
	if err := veleroplugin.ValidateObjectStoreConfigKeys(config,
		regionKey,
		s3URLKey,
		publicURLKey,
		kmsKeyIDKey,
		customerKeyEncryptionFileKey,
		s3ForcePathStyleKey,
		signatureVersionKey,
		credentialsFileKey,
		credentialProfileKey,
		serverSideEncryptionKey,
		insecureSkipTLSVerifyKey,
		enableSharedConfigKey,
	); err != nil {
		return err
	}

	var (
		region                    = config[regionKey]
		s3URL                     = config[s3URLKey]
		publicURL                 = config[publicURLKey]
		kmsKeyID                  = config[kmsKeyIDKey]
		customerKeyEncryptionFile = config[customerKeyEncryptionFileKey]
		s3ForcePathStyleVal       = config[s3ForcePathStyleKey]
		signatureVersion          = config[signatureVersionKey]
		credentialProfile         = config[credentialProfileKey]
		credentialsFile           = config[credentialsFileKey]
		serverSideEncryption      = config[serverSideEncryptionKey]
		insecureSkipTLSVerifyVal  = config[insecureSkipTLSVerifyKey]
		enableSharedConfig        = config[enableSharedConfigKey]

		// note that bucket is automatically added to the config map
		// by the server from the ObjectStorageProviderConfig so
		// doesn't need to be explicitly set by the user within
		// config.
		bucket                = config[bucketKey]
		caCert                = config[caCertKey]
		s3ForcePathStyle      bool
		insecureSkipTLSVerify bool
		err                   error
	)

	if s3ForcePathStyleVal != "" {
		if s3ForcePathStyle, err = strconv.ParseBool(s3ForcePathStyleVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", s3ForcePathStyleKey)
		}
	}

	// AWS (not an alternate S3-compatible API) and region not
	// explicitly specified: determine the bucket's region
	if s3URL == "" && region == "" {
		var err error

		region, err = GetBucketRegion(bucket)
		if err != nil {
			return err
		}
	}

	serverConfig, err := newAWSConfig(s3URL, region, s3ForcePathStyle)
	if err != nil {
		return err
	}

	if insecureSkipTLSVerifyVal != "" {
		if insecureSkipTLSVerify, err = strconv.ParseBool(insecureSkipTLSVerifyVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", insecureSkipTLSVerifyKey)
		}
	}

	if insecureSkipTLSVerify {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		serverConfig.HTTPClient = &http.Client{
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

	sessionOptions, err := newSessionOptions(*serverConfig, credentialProfile, caCert, credentialsFile, enableSharedConfig)
	if err != nil {
		return err
	}

	serverSession, err := getSession(sessionOptions)
	if err != nil {
		return err
	}

	o.s3 = s3.New(serverSession)
	o.s3Uploader = s3manager.NewUploader(serverSession)
	o.kmsKeyID = kmsKeyID
	o.serverSideEncryption = serverSideEncryption

	if customerKeyEncryptionFile != "" && kmsKeyID != "" {
		return errors.Wrapf(err, "you cannot use %s and %s at the same time", kmsKeyIDKey, customerKeyEncryptionFileKey)
	}

	if customerKeyEncryptionFile != "" {
		customerKey, err := readCustomerKey(customerKeyEncryptionFile)
		if err != nil {
			return err
		}
		o.sseCustomerKey = customerKey
	}

	if signatureVersion != "" {
		if !isValidSignatureVersion(signatureVersion) {
			return errors.Errorf("invalid signature version: %s", signatureVersion)
		}
		o.signatureVersion = signatureVersion
	}

	if publicURL != "" {
		publicConfig, err := newAWSConfig(publicURL, region, s3ForcePathStyle)
		if err != nil {
			return err
		}

		publicSessionOptions, err := newSessionOptions(*publicConfig, credentialProfile, caCert, credentialsFile, enableSharedConfig)
		if err != nil {
			return err
		}

		publicSession, err := getSession(publicSessionOptions)
		if err != nil {
			return err
		}
		o.preSignS3 = s3.New(publicSession)
	} else {
		o.preSignS3 = o.s3
	}

	return nil
}

func readCustomerKey(customerKeyEncryptionFile string) (string, error) {
	if _, err := os.Stat(customerKeyEncryptionFile); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Wrapf(err, "provided %s does not exist: %s", customerKeyEncryptionFileKey, customerKeyEncryptionFile)
		}
		return "", errors.Wrapf(err, "could not stat %s: %s", customerKeyEncryptionFileKey, customerKeyEncryptionFile)
	}

	fileHandle, err := os.Open(customerKeyEncryptionFile)
	if err != nil {
		return "", errors.Wrapf(err, "could not read %s: %s", customerKeyEncryptionFileKey, customerKeyEncryptionFile)
	}

	keyBytes := make([]byte, 32)
	nBytes, err := fileHandle.Read(keyBytes)
	if err != nil {
		return "", errors.Wrapf(err, "could not read %s: %s", customerKeyEncryptionFileKey, customerKeyEncryptionFile)
	}
	fileHandle.Close()

	if nBytes != 32 {
		return "", errors.Wrapf(err, "contents of %s (%s) are not exactly 32 bytes", customerKeyEncryptionFileKey, customerKeyEncryptionFile)
	}

	key := string(keyBytes)
	return key, nil
}

// newSessionOptions creates a session.Options with the given config and profile. If
// caCert and credentialsFile are provided, these will be used for the CustomCABundle
// and the credentials for the session.
func newSessionOptions(config aws.Config, profile string, caCert string, credentialsFile string, enableSharedConfig string) (session.Options, error) {
	sessionOptions := session.Options{Config: config, Profile: profile}

	if caCert != "" {
		sessionOptions.CustomCABundle = strings.NewReader(caCert)
	}

	if credentialsFile != "" {
		if _, err := os.Stat(credentialsFile); err != nil {
			if os.IsNotExist(err) {
				return session.Options{}, errors.Wrapf(err, "provided credentialsFile does not exist")
			}
			return session.Options{}, errors.Wrapf(err, "could not get credentialsFile info")
		}
		sessionOptions.SharedConfigFiles = []string{credentialsFile}

		if sharedConfig, berr := strconv.ParseBool(enableSharedConfig); sharedConfig && berr == nil {
			sessionOptions.SharedConfigState = session.SharedConfigEnable
		}
	}

	return sessionOptions, nil
}

func newAWSConfig(url, region string, forcePathStyle bool) (*aws.Config, error) {
	awsConfig := aws.NewConfig().
		WithRegion(region).
		WithS3ForcePathStyle(forcePathStyle)

	if url != "" {
		if !IsValidS3URLScheme(url) {
			return nil, errors.Errorf("Invalid s3 url %s, URL must be valid according to https://golang.org/pkg/net/url/#Parse and start with http:// or https://", url)
		}

		awsConfig = awsConfig.WithEndpointResolver(
			endpoints.ResolverFunc(func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
				if service == endpoints.S3ServiceID {
					return endpoints.ResolvedEndpoint{
						URL: url,
					}, nil
				}

				return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
			}),
		)
	}

	return awsConfig, nil
}

func (o *ObjectStore) PutObject(bucket, key string, body io.Reader) error {
	req := &s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   body,
	}

	switch {
	// if kmsKeyID is not empty, assume a server-side encryption (SSE)
	// algorithm of "aws:kms"
	case o.kmsKeyID != "":
		req.ServerSideEncryption = aws.String("aws:kms")
		req.SSEKMSKeyId = &o.kmsKeyID
	// if sseCustomerKey is not empty, assume SSE-C encryption with AES256 algorithm
	case o.sseCustomerKey != "":
		req.SSECustomerAlgorithm = aws.String("AES256")
		req.SSECustomerKey = &o.sseCustomerKey
	// otherwise, use the SSE algorithm specified, if any
	case o.serverSideEncryption != "":
		req.ServerSideEncryption = aws.String(o.serverSideEncryption)
	}

	_, err := o.s3Uploader.Upload(req)

	return errors.Wrapf(err, "error putting object %s", key)
}

const notFoundCode = "NotFound"

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	log := o.log.WithFields(
		logrus.Fields{
			"bucket": bucket,
			"key":    key,
		},
	)

	req := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if o.sseCustomerKey != "" {
		req.SSECustomerAlgorithm = aws.String("AES256")
		req.SSECustomerKey = &o.sseCustomerKey
	}

	log.Debug("Checking if object exists")
	if _, err := o.s3.HeadObject(req); err != nil {
		log.Debug("Checking for AWS specific error information")
		if aerr, ok := err.(awserr.Error); ok {
			log.WithFields(
				logrus.Fields{
					"code":    aerr.Code(),
					"message": aerr.Message(),
				},
			).Debugf("awserr.Error contents (origErr=%v)", aerr.OrigErr())

			// The code will be NotFound if the key doesn't exist.
			// See https://github.com/aws/aws-sdk-go/issues/1208 and https://github.com/aws/aws-sdk-go/pull/1213.
			log.Debugf("Checking for code=%s", notFoundCode)
			if aerr.Code() == notFoundCode {
				log.Debug("Object doesn't exist - got not found")
				return false, nil
			}
		}
		return false, errors.WithStack(err)
	}

	log.Debug("Object exists")
	return true, nil
}

func (o *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	req := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	if o.sseCustomerKey != "" {
		req.SSECustomerAlgorithm = aws.String("AES256")
		req.SSECustomerKey = &o.sseCustomerKey
	}

	res, err := o.s3.GetObject(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting object %s", key)
	}

	return res.Body, nil
}

func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	req := &s3.ListObjectsV2Input{
		Bucket:    &bucket,
		Prefix:    &prefix,
		Delimiter: &delimiter,
	}

	var ret []string
	err := o.s3.ListObjectsV2Pages(req, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, prefix := range page.CommonPrefixes {
			ret = append(ret, *prefix.Prefix)
		}
		return !lastPage
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return ret, nil
}

func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	req := &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	}

	var ret []string
	err := o.s3.ListObjectsV2Pages(req, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			ret = append(ret, *obj.Key)
		}
		return !lastPage
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	// ensure that returned objects are in a consistent order so that the deletion logic deletes the objects before
	// the pseudo-folder prefix object for s3 providers (such as Quobyte) that return the pseudo-folder as an object.
	// See https://github.com/vmware-tanzu/velero/pull/999
	sort.Sort(sort.Reverse(sort.StringSlice(ret)))

	return ret, nil
}

func (o *ObjectStore) DeleteObject(bucket, key string) error {
	req := &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	_, err := o.s3.DeleteObject(req)

	return errors.Wrapf(err, "error deleting object %s", key)
}

func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	req, _ := o.preSignS3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if o.signatureVersion == "1" {
		req.Handlers.Sign.Remove(v4.SignRequestHandler)
		req.Handlers.Sign.PushBackNamed(v1SignRequestHandler)
	}

	return req.Presign(ttl)
}
