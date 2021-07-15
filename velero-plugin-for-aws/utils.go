package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	corev1api "k8s.io/api/core/v1"
	kerror "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type JobStoreEntry struct {
	StartTimestamp  string `json:"start_timestamp,omitempty"`
	UpdateTimestamp string `json:"update_timestamp,omitempty"`
	JobType         string `json:"jobType,omitempty"`
}
type PvcInfo struct {
	Namespace              string `json:"namespace,omitempty"`
	DataTransferSnapshotId string `json:"data_transfer_snapshot_id,omitempty"`
	SnapshotId             string `json:"snapshot_id,omitempty"`
	VolumeId               string `json:"volume_id,omitempty"`
	Name                   string `json:"pv_name,omitempty"`
	PvName                 string `json:"pv_name,omitempty"`
	SnapshotType           string `json:"snapshot_type,omitempty"`
}
type PvcSnapshotProgressData struct {
	JobId            string   `json:"job_id,omitempty"`
	Message          string   `json:"message,omitempty"`
	State            string   `json:"state,omitempty"`
	SnapshotProgress int32    `json:"snapshot_progress,omitempty"`
	Pvc              *PvcInfo `json:"pvc,omitempty"`
}

const (
	filePermissions os.FileMode = 0755
	TimeFormat                  = "2006-01-02T15:04:05Z"
	// CasaBackupResource is the name of custom resource used to build the Velero GVR
	CasaBackupResource = "backups"
	// VeleroAPIGroup is the API group of Velero backup tool
	VeleroAPIGroup = "velero.io"
	// VeleroAPIVersion is the version of Veelro API's
	VeleroAPIVersion = "v1"
	// CasaVolumeSnapshotLocationResource is the name of custom resource used to build Velero VolumeSnapshotLocation GVR
	CasaVolumeSnapshotLocationResource = "volumesnapshotlocations"

	CloudCasaNamespace = "cloudcasa-io"

	// Name of configmap used to to report progress of snapshot
	SnapshotProgressUpdateConfigMapName = "cloudcasa-io-snapshot-updater"
)

var (
	// ResourceInventoryFilePaths is where the currently running jobs would be stored
	// Mutex to ensure serial access to go routines trying to update currently
	jobsStoreFilePath = "/scratch/jobStore.json"

	lockJobsStore sync.Mutex

	VolumeSnapshotLocationGVR = schema.GroupVersionResource{
		Group:    VeleroAPIGroup,
		Version:  VeleroAPIVersion,
		Resource: CasaVolumeSnapshotLocationResource,
	}
)

const (
	STORE_ADD StoreAction = iota
	STORE_UPDATE
	STORE_REMOVE
)

type StoreAction int

func (vs *VolumeSnapshotter) closeFile(fileHandle *os.File) error {
	err := fileHandle.Close()
	if err != nil {
		vs.log.Error(err, "Error in closing file")
	}
	return err
}

// SaveJobsStore saves a representation of v to the file at path.
func (vs *VolumeSnapshotter) saveJobsStore(v *map[string]interface{}) error {

	lockJobsStore.Lock()
	defer lockJobsStore.Unlock()

	fileHandle, err := os.OpenFile(jobsStoreFilePath, os.O_WRONLY|os.O_TRUNC, filePermissions)
	if err != nil {
		vs.log.Error(err, "JOBSTORE UPDATE- saveJobsStore() Failed to open job store", jobsStoreFilePath)
		return err
	}
	defer vs.closeFile(fileHandle)

	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		vs.log.Error(err, "JOBSTORE UPDATE- saveJobsStore(). Failed to encode file contents")
		return err
	}

	_, err = io.Copy(fileHandle, bytes.NewReader(b))
	if err != nil {
		vs.log.Error(err, "JOBSTORE UPDATE- saveJobsStore. Failed to copy centents to jobstore file")
		return err
	}

	return nil
}

// LoadJobsStore loads the file into v.
func (vs *VolumeSnapshotter) loadJobsStore(v *map[string]interface{}) error {
	lockJobsStore.Lock()
	defer lockJobsStore.Unlock()

	fileHandle, err := os.Open(jobsStoreFilePath)
	if err != nil {
		vs.log.Error(err, "JOBSTORE UPDATE- loadJobsStore() Failed to open job store", jobsStoreFilePath)
		return err
	}

	defer vs.closeFile(fileHandle)
	err = json.NewDecoder(fileHandle).Decode(v)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		vs.log.Error(err, "JOBSTORE UPDATE- loadJobsStore(). Failed to decode file contents")
		return err
	}

	return nil
}

//UpdateJobStore updates the Job Store
func (vs *VolumeSnapshotter) UpdateJobStore(timestamp string) error {

	m := make(map[string]interface{})

	if err := vs.loadJobsStore(&m); err != nil {
		vs.log.Error(err, "Failed to load the Jobs Store")
		return err
	}

	var updatedTimestamp bool
	for key, value := range m {
		jobEntry, ok := value.(map[string]interface{})
		if ok {

			if jobEntry["jobType"].(string) == CasaBackupResource {
				jobEntry["update_timestamp"] = timestamp
				m[key] = jobEntry
				updatedTimestamp = true
				// We expect there to be only 1 K8S_SNAP job
				// But, for whateever reason, there are more than 1,
				// we update the timestamps of all the backup jobs.
				// Hence, we do not "break" here
			}
		} else {
			vs.log.Error(fmt.Errorf("JOBSTORE UPDATE- Falied to unmarshall the job entry"), "Invalid jobstore found")
		}
	}

	if updatedTimestamp {
		err := vs.saveJobsStore(&m)
		if err != nil {
			vs.log.Error(err, "JOBSTORE UPDATE- Failed to save the Jobs Store")
			return err
		}
		vs.log.Info("JOBSTORE UPDATE- Updated timestamp for job in jobStore", "Contents of Jobstore", m)
	}

	return nil
}

// UpdateSnapshotProgress updates the configmap in order to relay the
// snapshot progress to KubeAgent
func (vs *VolumeSnapshotter) UpdateSnapshotProgress(
	volumeInfo *ec2.Volume,
	snapshotID string,
	tags map[string]string,
	percentageCompleteString string,
	state string,
	snapshotStateMessage string,
) error {
	vs.log.Info("Update Snapshot Progress - Starting to relay snapshot progress to KubAgent")
	// Fill in the PVC realted information
	var pvc = PvcInfo{}
	pvc.PvName = tags["velero.io/pv"]
	vs.log.Info("Update Snapshot Progress -", "PV Name", pvc.PvName)
	for _, tag := range volumeInfo.Tags {
		if *tag.Key == "kubernetes.io/created-for/pvc/name" {
			pvc.Name = *tag.Value
			vs.log.Info("Update Snapshot Progress -", "PVC Name", pvc.Name)
		}
		if *tag.Key == "kubernetes.io/created-for/pvc/namespace" {
			pvc.Namespace = *tag.Value
			vs.log.Info("Update Snapshot Progress -", "PVC Namespace", pvc.Namespace)
		}
	}
	pvc.SnapshotType = "NATIVE"
	pvc.SnapshotId = snapshotID
	pvc.VolumeId = *volumeInfo.VolumeId
	vs.log.Info("Update Snapshot Progress -", "PVC Payload", pvc)
	// Fill in Snapshot Progress related information
	var progress = PvcSnapshotProgressData{}
	progress.JobId = tags["velero.io/backup"]
	vs.log.Info("Update Snapshot Progress -", "Job ID", progress.JobId)
	progress.Message = snapshotStateMessage
	progress.State = state
	// Extract percentage from the string
	_, err := fmt.Sscanf(percentageCompleteString, "%d%%", &progress.SnapshotProgress)
	if err != nil {
		vs.log.Error(err, "Failed to convert percentage progress from string to int32")
	} else {
		vs.log.Info("Update Snapshot Progress -", "Percentage", progress.SnapshotProgress)
	}
	progress.Pvc = &pvc
	vs.log.Info("Update Snapshot Progress -", "Progress Payload", progress)

	// Prepare the paylod to be embedded into the configmap
	requestData := make(map[string][]byte)
	if requestData["snapshot_progress_payload"], err = json.Marshal(progress); err != nil {
		newErr := errors.Wrap(err, "Failed to marshal progress while creating the snapshot progress configmap")
		vs.log.Error(newErr, "JSON marshalling failed")
		return newErr
	}
	vs.log.Info("Update Snapshot Progress -", "Marsahlled the JSON payload")
	// create the configmap object.
	moverConfigMap := corev1api.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      SnapshotProgressUpdateConfigMapName,
			Namespace: CloudCasaNamespace,
		},
		BinaryData: requestData,
	}
	vs.log.Info("Update Snapshot Progress -", "Created the configmap object")
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		newErr := errors.Wrap(err, "Failed to create in-cluster config")
		vs.log.Error(newErr, "Failed to create in-cluster config")
		return newErr

	}
	vs.log.Info("Update Snapshot Progress -", "Created in-cluster config")
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		newErr := errors.Wrap(err, "Failed to create clientset")
		vs.log.Error(newErr, "Failed to create clientset")
		return newErr
	}
	vs.log.Info("Update Snapshot Progress -", "Created clientset")
	//Create or update the configmap
	var mcm *corev1api.ConfigMap
	if _, mErr := clientset.CoreV1().ConfigMaps(CloudCasaNamespace).Get(SnapshotProgressUpdateConfigMapName, v1.GetOptions{}); kerror.IsNotFound(mErr) {
		mcm, err = clientset.CoreV1().ConfigMaps(CloudCasaNamespace).Create(&moverConfigMap)
		if err != nil {
			newErr := errors.Wrap(err, "Failed to create configmap to report snapshotprogress")
			vs.log.Error(newErr, "Failed to create configmap")
			return newErr

		}
		vs.log.Info("Created configmap to report snapshot progress", "Configmap Name", mcm.GetName())
	} else {
		mcm, err = clientset.CoreV1().ConfigMaps(CloudCasaNamespace).Update(&moverConfigMap)
		if err != nil {
			newErr := errors.Wrap(err, "Failed to update configmap to report snapshotprogress")
			vs.log.Error(newErr, "Failed to update configmap")
			return newErr
		}
		vs.log.Info("Updated configmap to report snapshot progress", "Configmap Name", mcm.GetName())
	}
	vs.log.Info("finished relaying snapshot progress to KubeAgent")
	return nil
}

// DeleteSnapshotProgressConfigMap deletes the configmap used to report snapshot progress
func (vs *VolumeSnapshotter) DeleteSnapshotProgressConfigMap() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		vs.log.Error(errors.Wrap(err, "Failed to create in-cluster config"))
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		vs.log.Error(errors.Wrap(err, "Failed to create in-cluster clientset"))
	}
	err = clientset.CoreV1().ConfigMaps(CloudCasaNamespace).Delete(SnapshotProgressUpdateConfigMapName, &v1.DeleteOptions{})
	if err != nil {
		vs.log.Error(errors.Wrap(err, "Failed to delete configmap used to report snapshot progress"))
	} else {
		vs.log.Info("Deleted configmap used to report snapshot progress", "Configmap Name", SnapshotProgressUpdateConfigMapName)
	}
}
