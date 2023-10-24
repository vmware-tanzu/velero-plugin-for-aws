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
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

const (
	regionKey    = "region"
	ebsCSIDriver = "ebs.csi.aws.com"
)

// iopsVolumeTypes is a set of AWS EBS volume types for which IOPS should
// be captured during snapshot and provided when creating a new volume
// from snapshot.
var iopsVolumeTypes = sets.NewString("io1", "io2")

type VolumeSnapshotter struct {
	log logrus.FieldLogger
	ec2 *ec2.Client
}

func newVolumeSnapshotter(logger logrus.FieldLogger) *VolumeSnapshotter {
	return &VolumeSnapshotter{log: logger}
}

func (b *VolumeSnapshotter) Init(config map[string]string) error {
	if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config, regionKey, credentialProfileKey, credentialsFileKey, enableSharedConfigKey); err != nil {
		return err
	}

	region := config[regionKey]
	credentialProfile := config[credentialProfileKey]
	credentialsFile := config[credentialsFileKey]
	//	enableSharedConfig := config[enableSharedConfigKey]

	if region == "" {
		return errors.Errorf("missing %s in aws configuration", regionKey)
	}

	cfg, err := newAWSConfig(region, credentialProfile, credentialsFile, false, "")
	if err != nil {
		return errors.WithStack(err)
	}

	b.ec2 = ec2.NewFromConfig(cfg)

	return nil
}

func (b *VolumeSnapshotter) CreateVolumeFromSnapshot(snapshotID, volumeType, volumeAZ string, iops *int64) (volumeID string, err error) {
	// describe the snapshot so we can apply its tags to the volume
	descSnapInput := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{snapshotID},
	}
	descSnapOutput, err := b.ec2.DescribeSnapshots(context.Background(), descSnapInput)
	if err != nil {
		b.log.Infof("failed to describe snap shot: %v", err)

		return "", errors.WithStack(err)
	}

	if count := len(descSnapOutput.Snapshots); count != 1 {
		return "", errors.Errorf("expected 1 snapshot from DescribeSnapshots for %s, got %v", snapshotID, count)
	}

	// filter tags through getTagsForCluster() function in order to apply
	// proper ownership tags to restored volumes
	input := &ec2.CreateVolumeInput{
		SnapshotId:       &snapshotID,
		AvailabilityZone: &volumeAZ,
		VolumeType:       types.VolumeType(volumeType),
		Encrypted:        descSnapOutput.Snapshots[0].Encrypted,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVolume,
				Tags:         getTagsForCluster(descSnapOutput.Snapshots[0].Tags),
			},
		},
	}

	if iopsVolumeTypes.Has(volumeType) && iops != nil {
		iops32 := int32(*iops)
		input.Iops = &iops32
	}

	output, err := b.ec2.CreateVolume(context.Background(), input)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return *output.VolumeId, nil
}

func (b *VolumeSnapshotter) GetVolumeInfo(volumeID, volumeAZ string) (string, *int64, error) {
	volumeInfo, err := b.describeVolume(volumeID)
	if err != nil {
		return "", nil, err
	}

	var (
		volumeType string
		iops64     int64
	)

	volumeType = string(volumeInfo.VolumeType)

	if iopsVolumeTypes.Has(volumeType) && volumeInfo.Iops != nil {
		iops32 := volumeInfo.Iops
		iops64 = int64(*iops32)
	}

	return volumeType, &iops64, nil
}

func (b *VolumeSnapshotter) describeVolume(volumeID string) (types.Volume, error) {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	}

	output, err := b.ec2.DescribeVolumes(context.Background(), input)
	if err != nil {
		return types.Volume{}, errors.WithStack(err)
	}
	if count := len(output.Volumes); count != 1 {
		return types.Volume{}, errors.Errorf("Expected one volume from DescribeVolumes for volume ID %v, got %v", volumeID, count)
	}

	return output.Volumes[0], nil
}

func (b *VolumeSnapshotter) CreateSnapshot(volumeID, volumeAZ string, tags map[string]string) (string, error) {
	// describe the volume so we can copy its tags to the snapshot
	volumeInfo, err := b.describeVolume(volumeID)
	if err != nil {
		return "", err
	}

	res, err := b.ec2.CreateSnapshot(context.Background(), &ec2.CreateSnapshotInput{
		VolumeId: &volumeID,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSnapshot,
				Tags:         getTags(tags, volumeInfo.Tags),
			},
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	return *res.SnapshotId, nil
}

func getTagsForCluster(snapshotTags []types.Tag) []types.Tag {
	var result []types.Tag

	clusterName, haveAWSClusterNameEnvVar := os.LookupEnv("AWS_CLUSTER_NAME")

	if haveAWSClusterNameEnvVar {
		result = append(result, ec2Tag("kubernetes.io/cluster/"+clusterName, "owned"))
		result = append(result, ec2Tag("KubernetesCluster", clusterName))
	}

	for _, tag := range snapshotTags {
		if haveAWSClusterNameEnvVar && (strings.HasPrefix(*tag.Key, "kubernetes.io/cluster/") || *tag.Key == "KubernetesCluster") {
			// if the AWS_CLUSTER_NAME variable is found we want current cluster
			// to overwrite the old ownership on volumes
			continue
		}

		result = append(result, ec2Tag(*tag.Key, *tag.Value))
	}

	return result
}

func getTags(veleroTags map[string]string, volumeTags []types.Tag) []types.Tag {
	var result []types.Tag

	// set Velero-assigned tags
	for k, v := range veleroTags {
		result = append(result, ec2Tag(k, v))
	}

	// copy tags from volume to snapshot
	for _, tag := range volumeTags {
		// we want current Velero-assigned tags to overwrite any older versions
		// of them that may exist due to prior snapshots/restores
		if _, found := veleroTags[*tag.Key]; found {
			continue
		}

		result = append(result, ec2Tag(*tag.Key, *tag.Value))
	}

	return result
}

func ec2Tag(key, val string) types.Tag {
	return types.Tag{Key: &key, Value: &val}
}

func (b *VolumeSnapshotter) DeleteSnapshot(snapshotID string) error {
	input := &ec2.DeleteSnapshotInput{
		SnapshotId: &snapshotID,
	}
	_, err := b.ec2.DeleteSnapshot(context.Background(), input)

	// if it's a NotFound error, we don't need to return an error
	// since the snapshot is not there.
	// see https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if "InvalidSnapshot.NotFound" == apiErr.ErrorCode() {
			return nil
		}
	}

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

var ebsVolumeIDRegex = regexp.MustCompile("vol-.*")

func (b *VolumeSnapshotter) GetVolumeID(unstructuredPV runtime.Unstructured) (string, error) {
	pv := new(v1.PersistentVolume)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPV.UnstructuredContent(), pv); err != nil {
		return "", errors.WithStack(err)
	}
	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == ebsCSIDriver {
			return ebsVolumeIDRegex.FindString(pv.Spec.CSI.VolumeHandle), nil
		}
		b.log.Infof("Unable to handle CSI driver: %s", driver)
	}

	if pv.Spec.AWSElasticBlockStore != nil {
		if pv.Spec.AWSElasticBlockStore.VolumeID == "" {
			return "", errors.New("spec.awsElasticBlockStore.volumeID not found")
		}
		return ebsVolumeIDRegex.FindString(pv.Spec.AWSElasticBlockStore.VolumeID), nil
	}

	return "", nil
}

func (b *VolumeSnapshotter) SetVolumeID(unstructuredPV runtime.Unstructured, volumeID string) (runtime.Unstructured, error) {
	pv := new(v1.PersistentVolume)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPV.UnstructuredContent(), pv); err != nil {
		return nil, errors.WithStack(err)
	}
	if pv.Spec.CSI != nil {
		// PV is provisioned by CSI driver
		driver := pv.Spec.CSI.Driver
		if driver == ebsCSIDriver {
			pv.Spec.CSI.VolumeHandle = volumeID
		} else {
			return nil, fmt.Errorf("unable to handle CSI driver: %s", driver)
		}
	} else if pv.Spec.AWSElasticBlockStore != nil {
		// PV is provisioned by in-tree driver
		pvFailureDomainZone := pv.Labels["failure-domain.beta.kubernetes.io/zone"]
		if len(pvFailureDomainZone) > 0 {
			pv.Spec.AWSElasticBlockStore.VolumeID = fmt.Sprintf("aws://%s/%s", pvFailureDomainZone, volumeID)
		} else {
			pv.Spec.AWSElasticBlockStore.VolumeID = volumeID
		}
	} else {
		return nil, errors.New("spec.csi and spec.awsElasticBlockStore not found")
	}

	res, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pv)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &unstructured.Unstructured{Object: res}, nil
}
