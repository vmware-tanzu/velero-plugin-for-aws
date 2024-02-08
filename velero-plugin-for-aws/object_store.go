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
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

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
	taggingKey                   = "tagging"
	checksumAlgKey               = "checksumAlgorithm"
)

type s3Interface interface {
	HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type s3PresignInterface interface {
	PresignGetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(options *s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type ObjectStore struct {
	log                  logrus.FieldLogger
	s3                   s3Interface
	preSignS3            s3PresignInterface
	s3Uploader           *manager.Uploader
	kmsKeyID             string
	sseCustomerKey       string
	signatureVersion     string
	serverSideEncryption string
	tagging              string
	checksumAlg          string
}

func newObjectStore(logger logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: logger}
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
		taggingKey,
		checksumAlgKey,
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
		credentialProfile         = config[credentialProfileKey]
		credentialsFile           = config[credentialsFileKey]
		serverSideEncryption      = config[serverSideEncryptionKey]
		insecureSkipTLSVerifyVal  = config[insecureSkipTLSVerifyKey]
		tagging                   = config[taggingKey]
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
		region, err = GetBucketRegion(bucket, s3ForcePathStyle)
		if err != nil {
			o.log.Errorf("Failed to get bucket region, bucket: %s, error: %v", bucket, err)
			return err
		}
	}

	if insecureSkipTLSVerifyVal != "" {
		if insecureSkipTLSVerify, err = strconv.ParseBool(insecureSkipTLSVerifyVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", insecureSkipTLSVerifyKey)
		}
	}

	cfg, err := newAWSConfig(region, credentialProfile, credentialsFile, insecureSkipTLSVerify, caCert)
	if err != nil {
		return errors.WithStack(err)
	}

	client, err := newS3Client(cfg, s3URL, s3ForcePathStyle)
	if err != nil {
		return errors.WithStack(err)
	}
	o.s3 = client
	o.s3Uploader = manager.NewUploader(client)
	o.kmsKeyID = kmsKeyID
	o.serverSideEncryption = serverSideEncryption
	o.tagging = tagging

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

	if publicURL != "" {
		publicClient, err := newS3Client(cfg, publicURL, s3ForcePathStyle)
		if err != nil {
			return err
		}

		o.preSignS3 = s3.NewPresignClient(publicClient)
	} else {
		o.preSignS3 = s3.NewPresignClient(client)
	}
	if tagging != "" {
		err = CheckTags(tagging)
		if err != nil {
			return err
		}
	}
	if alg, ok := config[checksumAlgKey]; ok {
		if !validChecksumAlg(alg) {
			return errors.Errorf("invalid checksum algorithm: %s", alg)
		}
		o.checksumAlg = alg
	} else {
		o.checksumAlg = string(types.ChecksumAlgorithmCrc32)
	}
	return nil
}

func validChecksumAlg(alg string) bool {
	return alg == string(types.ChecksumAlgorithmCrc32) || alg == string(types.ChecksumAlgorithmCrc32c) ||
		alg == string(types.ChecksumAlgorithmSha1) || alg == string(types.ChecksumAlgorithmSha256) || alg == ""
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

func (o *ObjectStore) PutObject(bucket, key string, body io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:  aws.String(bucket),
		Key:     aws.String(key),
		Body:    body,
		Tagging: aws.String(o.tagging),
	}

	switch {
	// if kmsKeyID is not empty, assume a server-side encryption (SSE)
	// algorithm of "aws:kms"
	case o.kmsKeyID != "":
		input.ServerSideEncryption = "aws:kms"
		input.SSEKMSKeyId = &o.kmsKeyID
	// if sseCustomerKey is not empty, assume SSE-C encryption with AES256 algorithm
	case o.sseCustomerKey != "":
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
	// otherwise, use the SSE algorithm specified, if any
	case o.serverSideEncryption != "":
		input.ServerSideEncryption = types.ServerSideEncryption(o.serverSideEncryption)
	}

	if o.checksumAlg != "" {
		input.ChecksumAlgorithm = types.ChecksumAlgorithm(o.checksumAlg)
	}

	_, err := o.s3Uploader.Upload(context.Background(), input)

	return errors.Wrapf(err, "error putting object %s", key)
}

// ObjectExists checks if there is an object with the given key in the object storage bucket.
func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	log := o.log.WithFields(
		logrus.Fields{
			"bucket": bucket,
			"key":    key,
		},
	)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if o.sseCustomerKey != "" {
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
	}

	log.Debug("Checking if object exists")
	if _, err := o.s3.HeadObject(context.Background(), input); err != nil {
		log.Debug("Checking for AWS specific error information")
		var ne *types.NotFound
		if errors.As(err, &ne) {
			log.WithFields(
				logrus.Fields{
					"code":    ne.ErrorCode(),
					"message": ne.ErrorMessage(),
				},
			).Debug("Object doesn't exist - got not found")
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	log.Debug("Object exists")
	return true, nil
}

func (o *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if o.sseCustomerKey != "" {
		input.SSECustomerAlgorithm = aws.String("AES256")
		input.SSECustomerKey = &o.sseCustomerKey
	}

	output, err := o.s3.GetObject(context.Background(), input)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting object %s", key)
	}

	return output.Body, nil
}

func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}
	var ret []string
	p := s3.NewListObjectsV2Paginator(o.s3, input)
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, prefix := range page.CommonPrefixes {
			ret = append(ret, *prefix.Prefix)
		}
	}
	return ret, nil
}

func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	var ret []string
	p := s3.NewListObjectsV2Paginator(o.s3, input)
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, obj := range page.Contents {
			ret = append(ret, *obj.Key)
		}
	}
	// ensure that returned objects are in a consistent order so that the deletion logic deletes the objects before
	// the pseudo-folder prefix object for s3 providers (such as Quobyte) that return the pseudo-folder as an object.
	// See https://github.com/vmware-tanzu/velero/pull/999
	sort.Sort(sort.Reverse(sort.StringSlice(ret)))

	return ret, nil
}

func (o *ObjectStore) DeleteObject(bucket, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := o.s3.DeleteObject(context.Background(), input)

	return errors.Wrapf(err, "error deleting object %s", key)
}

func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	req, err := o.preSignS3.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})

	if err != nil {
		return "", errors.WithStack(err)
	}
	return req.URL, nil
}
