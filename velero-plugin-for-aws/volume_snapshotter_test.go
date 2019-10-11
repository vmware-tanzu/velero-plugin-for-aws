/*
Copyright 2017, 2019 the Velero contributors.

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
	"os"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetVolumeID(t *testing.T) {
	b := &VolumeSnapshotter{}

	pv := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}

	// missing spec.awsElasticBlockStore -> no error
	volumeID, err := b.GetVolumeID(pv)
	require.NoError(t, err)
	assert.Equal(t, "", volumeID)

	// missing spec.awsElasticBlockStore.volumeID -> error
	aws := map[string]interface{}{}
	pv.Object["spec"] = map[string]interface{}{
		"awsElasticBlockStore": aws,
	}
	volumeID, err = b.GetVolumeID(pv)
	assert.Error(t, err)
	assert.Equal(t, "", volumeID)

	// regex miss
	aws["volumeID"] = "foo"
	volumeID, err = b.GetVolumeID(pv)
	assert.NoError(t, err)
	assert.Equal(t, "", volumeID)

	// regex match 1
	aws["volumeID"] = "aws://us-east-1c/vol-abc123"
	volumeID, err = b.GetVolumeID(pv)
	assert.NoError(t, err)
	assert.Equal(t, "vol-abc123", volumeID)

	// regex match 2
	aws["volumeID"] = "vol-abc123"
	volumeID, err = b.GetVolumeID(pv)
	assert.NoError(t, err)
	assert.Equal(t, "vol-abc123", volumeID)
}

func TestSetVolumeID(t *testing.T) {
	b := &VolumeSnapshotter{}

	pv := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}

	// missing spec.awsElasticBlockStore -> error
	updatedPV, err := b.SetVolumeID(pv, "vol-updated")
	require.Error(t, err)

	// happy path
	aws := map[string]interface{}{}
	pv.Object["spec"] = map[string]interface{}{
		"awsElasticBlockStore": aws,
	}

	labels := map[string]interface{}{
		"failure-domain.beta.kubernetes.io/zone": "us-east-1a",
	}

	pv.Object["metadata"] = map[string]interface{}{
		"labels": labels,
	}

	updatedPV, err = b.SetVolumeID(pv, "vol-updated")

	require.NoError(t, err)

	res := new(v1.PersistentVolume)
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(updatedPV.UnstructuredContent(), res))
	require.NotNil(t, res.Spec.AWSElasticBlockStore)
	assert.Equal(t, "aws://us-east-1a/vol-updated", res.Spec.AWSElasticBlockStore.VolumeID)
}

func TestSetVolumeIDNoZone(t *testing.T) {
	b := &VolumeSnapshotter{}

	pv := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}

	// missing spec.awsElasticBlockStore -> error
	updatedPV, err := b.SetVolumeID(pv, "vol-updated")
	require.Error(t, err)

	// happy path
	aws := map[string]interface{}{}
	pv.Object["spec"] = map[string]interface{}{
		"awsElasticBlockStore": aws,
	}

	updatedPV, err = b.SetVolumeID(pv, "vol-updated")

	require.NoError(t, err)

	res := new(v1.PersistentVolume)
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(updatedPV.UnstructuredContent(), res))
	require.NotNil(t, res.Spec.AWSElasticBlockStore)
	assert.Equal(t, "vol-updated", res.Spec.AWSElasticBlockStore.VolumeID)
}

func TestGetTagsForCluster(t *testing.T) {
	tests := []struct {
		name         string
		isNameSet    bool
		snapshotTags []*ec2.Tag
		expected     []*ec2.Tag
	}{
		{
			name:         "degenerate case (no tags)",
			isNameSet:    false,
			snapshotTags: nil,
			expected:     nil,
		},
		{
			name:      "cluster tags exist and remain set",
			isNameSet: false,
			snapshotTags: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "old-cluster"),
				ec2Tag("kubernetes.io/cluster/old-cluster", "owned"),
				ec2Tag("aws-key", "aws-val"),
			},
			expected: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "old-cluster"),
				ec2Tag("kubernetes.io/cluster/old-cluster", "owned"),
				ec2Tag("aws-key", "aws-val"),
			},
		},
		{
			name:         "cluster tags only get applied",
			isNameSet:    true,
			snapshotTags: nil,
			expected: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "current-cluster"),
				ec2Tag("kubernetes.io/cluster/current-cluster", "owned"),
			},
		},
		{
			name:         "non-overlaping cluster and snapshot tags both get applied",
			isNameSet:    true,
			snapshotTags: []*ec2.Tag{ec2Tag("aws-key", "aws-val")},
			expected: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "current-cluster"),
				ec2Tag("kubernetes.io/cluster/current-cluster", "owned"),
				ec2Tag("aws-key", "aws-val"),
			},
		},
		{name: "overlaping cluster tags, current cluster tags take precedence",
			isNameSet: true,
			snapshotTags: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "old-name"),
				ec2Tag("kubernetes.io/cluster/old-name", "owned"),
				ec2Tag("aws-key", "aws-val"),
			},
			expected: []*ec2.Tag{
				ec2Tag("KubernetesCluster", "current-cluster"),
				ec2Tag("kubernetes.io/cluster/current-cluster", "owned"),
				ec2Tag("aws-key", "aws-val"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.isNameSet {
				os.Setenv("AWS_CLUSTER_NAME", "current-cluster")
			}
			res := getTagsForCluster(test.snapshotTags)

			sort.Slice(res, func(i, j int) bool {
				return *res[i].Key < *res[j].Key
			})

			sort.Slice(test.expected, func(i, j int) bool {
				return *test.expected[i].Key < *test.expected[j].Key
			})

			assert.Equal(t, test.expected, res)

			if test.isNameSet {
				os.Unsetenv("AWS_CLUSTER_NAME")
			}
		})
	}
}

func TestGetTags(t *testing.T) {
	tests := []struct {
		name       string
		veleroTags map[string]string
		volumeTags []*ec2.Tag
		expected   []*ec2.Tag
	}{
		{
			name:       "degenerate case (no tags)",
			veleroTags: nil,
			volumeTags: nil,
			expected:   nil,
		},
		{
			name: "velero tags only get applied",
			veleroTags: map[string]string{
				"velero-key1": "velero-val1",
				"velero-key2": "velero-val2",
			},
			volumeTags: nil,
			expected: []*ec2.Tag{
				ec2Tag("velero-key1", "velero-val1"),
				ec2Tag("velero-key2", "velero-val2"),
			},
		},
		{
			name:       "volume tags only get applied",
			veleroTags: nil,
			volumeTags: []*ec2.Tag{
				ec2Tag("aws-key1", "aws-val1"),
				ec2Tag("aws-key2", "aws-val2"),
			},
			expected: []*ec2.Tag{
				ec2Tag("aws-key1", "aws-val1"),
				ec2Tag("aws-key2", "aws-val2"),
			},
		},
		{
			name:       "non-overlapping velero and volume tags both get applied",
			veleroTags: map[string]string{"velero-key": "velero-val"},
			volumeTags: []*ec2.Tag{ec2Tag("aws-key", "aws-val")},
			expected: []*ec2.Tag{
				ec2Tag("velero-key", "velero-val"),
				ec2Tag("aws-key", "aws-val"),
			},
		},
		{
			name: "when tags overlap, velero tags take precedence",
			veleroTags: map[string]string{
				"velero-key":      "velero-val",
				"overlapping-key": "velero-val",
			},
			volumeTags: []*ec2.Tag{
				ec2Tag("aws-key", "aws-val"),
				ec2Tag("overlapping-key", "aws-val"),
			},
			expected: []*ec2.Tag{
				ec2Tag("velero-key", "velero-val"),
				ec2Tag("overlapping-key", "velero-val"),
				ec2Tag("aws-key", "aws-val"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := getTags(test.veleroTags, test.volumeTags)

			sort.Slice(res, func(i, j int) bool {
				return *res[i].Key < *res[j].Key
			})

			sort.Slice(test.expected, func(i, j int) bool {
				return *test.expected[i].Key < *test.expected[j].Key
			})

			assert.Equal(t, test.expected, res)
		})
	}
}
