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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS3URL(t *testing.T) {
	assert.True(t, IsValidS3URLScheme("http://foo"))
	assert.True(t, IsValidS3URLScheme("https://foo"))
	assert.False(t, IsValidS3URLScheme("httpd://foo"))
	assert.False(t, IsValidS3URLScheme(""))
}

func TestS3Tags(t *testing.T) {
	assert.Error(t, CheckTags("96FrFmTtJcBkEYEVtS3Bxrv2E37KG9m3M9CJqbtVCw7gy4UBEvpBC4h6xdV7FUBag7XeZhccvQuY8AgERdeWafBZRR7NRb8BnA6CkcqDHPpPPFpwLzXenjxZmeRK6J9hty=Value1&Key2=Value2&Key3=Value3"))
	assert.Error(t, CheckTags("Key1=Value1&Key2=tka2MYGaFegMVkdm5nC58D46dyXCDbKcXnCZNrCHyS6s8TtMacs9HFXpGCNr2PVntCHArkKbgyYntVhBn2AAJXdKkvA9jdRrc4vCsYzCSZ4ZhCR7PBaKgMMdTtz93jRZNNFJcAqzrybDqCEmtKfFj3MdxLSvjej9tqP8bUt66449ZCbPuk8b7ASrYkPf6fQVYXM9rbmtyzWbhxtYdZs7nUaE4pQqtEfggqSEGbNaNWSf6x6vjUVJ2fAsZNfwKMxUwe"))
	assert.Error(t, CheckTags("Key1=Value1&Key2=Value2&Key3-Value3&Key4=Value4&Key5=Value5&Key6=Value6&Key7=Value7&Key8=Value8&Key9=Value9&key10=Value10&Key11=Value11"))
	assert.Nil(t, CheckTags("Key1=Value1&Key2=Value2"))
	assert.ErrorIs(t, CheckTags("Key1=Value1&Key2=Value2&Key3=Value3"), nil)
}
