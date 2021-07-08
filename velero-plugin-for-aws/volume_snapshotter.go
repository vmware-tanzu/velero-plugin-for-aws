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
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

const regionKey = "region"

// iopsVolumeTypes is a set of AWS EBS volume types for which IOPS should
// be captured during snapshot and provided when creating a new volume
// from snapshot.
var iopsVolumeTypes = sets.NewString("io1")

type VolumeSnapshotter struct {
	log    logrus.FieldLogger
	ec2    *ec2.EC2
	config map[string]string
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

func newVolumeSnapshotter(logger logrus.FieldLogger) *VolumeSnapshotter {
	return &VolumeSnapshotter{log: logger}
}

func (b *VolumeSnapshotter) Init(config map[string]string) error {
	if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config, regionKey, credentialProfileKey); err != nil {
		return err
	}

	region := config[regionKey]
	credentialProfile := config[credentialProfileKey]
	if region == "" {
		return errors.Errorf("missing %s in aws configuration", regionKey)
	}

	awsConfig := aws.NewConfig().WithRegion(region)

	sessionOptions := session.Options{Config: *awsConfig, Profile: credentialProfile}
	sess, err := getSession(sessionOptions)
	if err != nil {
		return err
	}

	b.ec2 = ec2.New(sess)
	b.config = config

	return nil
}

func (b *VolumeSnapshotter) CreateVolumeFromSnapshot(snapshotID, volumeType, volumeAZ string, iops *int64) (volumeID string, err error) {
	// describe the snapshot so we can apply its tags to the volume
	snapReq := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{&snapshotID},
	}

	snapRes, err := b.ec2.DescribeSnapshots(snapReq)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if count := len(snapRes.Snapshots); count != 1 {
		return "", errors.Errorf("expected 1 snapshot from DescribeSnapshots for %s, got %v", snapshotID, count)
	}

	// filter tags through getTagsForCluster() function in order to apply
	// proper ownership tags to restored volumes
	req := &ec2.CreateVolumeInput{
		SnapshotId:       &snapshotID,
		AvailabilityZone: &volumeAZ,
		VolumeType:       &volumeType,
		Encrypted:        snapRes.Snapshots[0].Encrypted,
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeVolume),
				Tags:         getTagsForCluster(snapRes.Snapshots[0].Tags),
			},
		},
	}

	if iopsVolumeTypes.Has(volumeType) && iops != nil {
		req.Iops = iops
	}

	res, err := b.ec2.CreateVolume(req)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return *res.VolumeId, nil
}

func (b *VolumeSnapshotter) GetVolumeInfo(volumeID, volumeAZ string) (string, *int64, error) {
	volumeInfo, err := b.describeVolume(volumeID)
	if err != nil {
		return "", nil, err
	}

	var (
		volumeType string
		iops       *int64
	)

	if volumeInfo.VolumeType != nil {
		volumeType = *volumeInfo.VolumeType
	}

	if iopsVolumeTypes.Has(volumeType) && volumeInfo.Iops != nil {
		iops = volumeInfo.Iops
	}

	return volumeType, iops, nil
}

func (b *VolumeSnapshotter) describeVolume(volumeID string) (*ec2.Volume, error) {
	req := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{&volumeID},
	}

	res, err := b.ec2.DescribeVolumes(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if count := len(res.Volumes); count != 1 {
		return nil, errors.Errorf("Expected one volume from DescribeVolumes for volume ID %v, got %v", volumeID, count)
	}

	return res.Volumes[0], nil
}

func (b *VolumeSnapshotter) CreateSnapshot(volumeID, volumeAZ string, tags map[string]string) (string, error) {
	// describe the volume so we can copy its tags to the snapshot
	volumeInfo, err := b.describeVolume(volumeID)
	if err != nil {
		return "", err
	}

	res, err := b.ec2.CreateSnapshot(&ec2.CreateSnapshotInput{
		VolumeId: &volumeID,
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeSnapshot),
				Tags:         getTags(tags, volumeInfo.Tags),
			},
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	snapshotID := *res.SnapshotId

	// wait for the snapshot to be completed
	var previousProgress string
	refreshCredsSec := 3300
	t := 0

	for {
		currentTimeString := time.Now().Format(TimeFormat)
		err := b.UpdateJobStore(currentTimeString)
		if err != nil {
			b.log.Error(err, "<JOBSTORE UPDATE> Failed to updated timestamp in jobstore. Continuing...")
		}
		if t != 0 && t%refreshCredsSec == 0 {
			// more than 45 minutes have passed for the temporary credentials so create a new session
			err := b.Init(b.config)
			if err != nil {
				return "", errors.WithStack(err)
			}
		}

		snapRes, err := b.ec2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
			SnapshotIds: []*string{res.SnapshotId},
		})
		if err != nil {
			return "", errors.WithStack(err)
		}

		if count := len(snapRes.Snapshots); count != 1 {
			return "", errors.Errorf("expected 1 snapshot from DescribeSnapshots for %s, got %v", snapshotID, count)
		}

		if *snapRes.Snapshots[0].State == "completed" {
			break
		}

		if t == 3600 {
			// set progress after 1 hour has passed
			previousProgress = *snapRes.Snapshots[0].Progress
		} else if t > 3600 && t%300 == 0 {
			if previousProgress == *snapRes.Snapshots[0].Progress {
				return "", errors.Errorf("EBS volume snapshot %s progress has been stuck on %s for 1 hour", snapshotID, previousProgress)
			}
			previousProgress = *snapRes.Snapshots[0].Progress
		}

		t += 15
		time.Sleep(15 * time.Second)
	}

	return snapshotID, nil
}

func getTagsForCluster(snapshotTags []*ec2.Tag) []*ec2.Tag {
	var result []*ec2.Tag

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

func getTags(veleroTags map[string]string, volumeTags []*ec2.Tag) []*ec2.Tag {
	var result []*ec2.Tag

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

func ec2Tag(key, val string) *ec2.Tag {
	return &ec2.Tag{Key: &key, Value: &val}
}

func (b *VolumeSnapshotter) DeleteSnapshot(snapshotID string) error {
	req := &ec2.DeleteSnapshotInput{
		SnapshotId: &snapshotID,
	}

	_, err := b.ec2.DeleteSnapshot(req)

	// if it's a NotFound error, we don't need to return an error
	// since the snapshot is not there.
	// see https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidSnapshot.NotFound" {
		return nil
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

	if pv.Spec.AWSElasticBlockStore == nil {
		return "", nil
	}

	if pv.Spec.AWSElasticBlockStore.VolumeID == "" {
		return "", errors.New("spec.awsElasticBlockStore.volumeID not found")
	}

	return ebsVolumeIDRegex.FindString(pv.Spec.AWSElasticBlockStore.VolumeID), nil
}

func (b *VolumeSnapshotter) SetVolumeID(unstructuredPV runtime.Unstructured, volumeID string) (runtime.Unstructured, error) {
	pv := new(v1.PersistentVolume)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredPV.UnstructuredContent(), pv); err != nil {
		return nil, errors.WithStack(err)
	}

	if pv.Spec.AWSElasticBlockStore == nil {
		return nil, errors.New("spec.awsElasticBlockStore not found")
	}

	pvFailureDomainZone := pv.Labels["failure-domain.beta.kubernetes.io/zone"]

	if len(pvFailureDomainZone) > 0 {
		pv.Spec.AWSElasticBlockStore.VolumeID = fmt.Sprintf("aws://%s/%s", pvFailureDomainZone, volumeID)
	} else {
		pv.Spec.AWSElasticBlockStore.VolumeID = volumeID
	}

	res, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pv)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &unstructured.Unstructured{Object: res}, nil
}
