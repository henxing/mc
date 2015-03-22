// Original license //
// ---------------- //

/*
Copyright 2011 Google Inc.

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

// All other modifications and improvements //
// ---------------------------------------- //

/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package s3 implements a generic Amazon S3 client
package s3

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"encoding/xml"

	"github.com/minio-io/mc/pkg/client"
)

// Total max object list
const (
	MaxKeys = 1000
)

type listBucketResults struct {
	Contents       []*client.Item
	IsTruncated    bool
	MaxKeys        int
	Name           string // bucket name
	Marker         string
	Delimiter      string
	Prefix         string
	CommonPrefixes []*client.Prefix
}

type s3Client struct {
	*client.Meta
}

// GetNewClient returns an initialized s3Client structure.
func GetNewClient(auth *client.Auth, u *url.URL, transport http.RoundTripper) client.Client {
	return &s3Client{&client.Meta{
		Auth:      auth,
		Transport: GetNewTraceTransport(s3Verify{}, transport),
		URL:       u,
	}}
}

// bucketURL returns the URL prefix of the bucket, with trailing slash
func (c *s3Client) bucketURL(bucket string) string {
	var url string
	if IsValidBucket(bucket) && !strings.Contains(bucket, ".") {
		// if localhost use PathStyle
		if strings.Contains(c.URL.Host, "localhost") || strings.Contains(c.URL.Host, "127.0.0.1") {
			return fmt.Sprintf("%s://%s/%s", c.URL.Scheme, c.URL.Host, bucket)
		}
		// Verify if its ip address, use PathStyle
		host, _, _ := net.SplitHostPort(c.URL.Host)
		if net.ParseIP(host) != nil {
			return fmt.Sprintf("%s://%s/%s", c.URL.Scheme, c.URL.Host, bucket)
		}
		// For DNS hostname or amazonaws.com use subdomain style
		url = fmt.Sprintf("%s://%s.%s/", c.URL.Scheme, bucket, c.URL.Host)
	}
	return url
}

func (c *s3Client) keyURL(bucket, key string) string {
	url := c.bucketURL(bucket)
	if strings.Contains(c.URL.Host, "localhost") || strings.Contains(c.URL.Host, "127.0.0.1") {
		return url + "/" + key
	}
	host, _, _ := net.SplitHostPort(c.URL.Host)
	if net.ParseIP(host) != nil {
		return url + "/" + key
	}
	return url + key
}

func newReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(fmt.Sprintf("s3 client; invalid URL: %v", err))
	}
	req.Header.Set("User-Agent", "Minio s3Client")
	return req
}

func parseListAllMyBuckets(r io.Reader) ([]*client.Bucket, error) {
	type allMyBuckets struct {
		Buckets struct {
			Bucket []*client.Bucket
		}
	}
	var res allMyBuckets
	if err := xml.NewDecoder(r).Decode(&res); err != nil {
		return nil, err
	}
	return res.Buckets.Bucket, nil
}

// IsValidBucket reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucket(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match == true {
		return false
	}
	// We don't support buckets with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucket)
	return match
}
