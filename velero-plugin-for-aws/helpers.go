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
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

// GetBucketRegion returns the AWS region that a bucket is in, or an error
// if the region cannot be determined.
func GetBucketRegion(bucket string, usePathStyle bool) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return "", errors.WithStack(err)
	}
	client := s3.NewFromConfig(cfg)
	region, err := s3manager.GetBucketRegion(context.Background(), client, bucket, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})
	if err != nil {
		return "", err
	}
	if region == "" {
		return "", errors.New("unable to determine bucket's region")
	}
	return region, nil
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

func CheckTags(tagging string) error {
	tags := strings.Split(tagging, "&")
	if len(tags) == 1 {
		return errors.New("Tags are not seperated with an &")
	}
	for c, j := range tags {
		if c > 9 {
			return errors.New("Aws S3 allows only ten tags per object")
		}
		tg := strings.Split(j, "=")
		if len(tg) != 2 {
			return errors.New("invalid tags provided")
		} else {
			if len([]rune(tg[0])) > 128 {
				return errors.New("An S3 tag key can not be more than 128 Unicode characters in length")
			} else {
				if len([]rune(tg[1])) > 248 {
					return errors.New("An S3 tag values can not be more 256 Unicode characters in length")
				}
			}
		}
	}
	return nil
}
