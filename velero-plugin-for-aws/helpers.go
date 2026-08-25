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
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

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

// stripDefaultPort removes the port from an S3 endpoint URL when it is the
// default for the URL's scheme, and returns the URL unchanged otherwise.
//
// The SigV4 signer omits a scheme-default port from the host header it signs,
// but leaves the port in the request URL. That is invisible for a request the
// plugin sends itself, because the signer rewrites the outgoing Host header to
// match. It is not invisible in a presigned URL: whoever fetches it derives the
// Host from the URL, so the server hashes "host:443" against a signature
// computed over "host". Amazon S3 normalizes the Host header before verifying
// and hides the discrepancy, but S3-compatible backends generally do not and
// answer SignatureDoesNotMatch.
//
// See https://github.com/velero-io/velero/issues/10114
func stripDefaultPort(s3URL string) string {
	u, err := url.Parse(s3URL)
	if err != nil {
		return s3URL
	}

	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		// Trimming the suffix rather than using u.Hostname() keeps the brackets
		// around an IPv6 literal.
		u.Host = strings.TrimSuffix(u.Host, ":"+port)
		return u.String()
	}

	return s3URL
}

func CheckTags(tagging string) error {
	tags := strings.Split(tagging, "&")
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
				if len([]rune(tg[1])) > 256 {
					return errors.New("An S3 tag values can not be more 256 Unicode characters in length")
				}
			}
		}
	}
	return nil
}
