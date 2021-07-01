package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type JobStoreEntry struct {
	StartTimestamp  string `json:"start_timestamp,omitempty"`
	UpdateTimestamp string `json:"update_timestamp,omitempty"`
	JobType         string `json:"jobType,omitempty"`
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
		return err
	}
	defer vs.closeFile(fileHandle)

	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}

	_, err = io.Copy(fileHandle, bytes.NewReader(b))
	return err
}

// LoadJobsStore loads the file into v.
func (vs *VolumeSnapshotter) loadJobsStore(v *map[string]interface{}) error {
	lockJobsStore.Lock()
	defer lockJobsStore.Unlock()
	fileHandle, err := os.Open(jobsStoreFilePath)
	if err != nil {
		return err
	}
	defer vs.closeFile(fileHandle)
	err = json.NewDecoder(fileHandle).Decode(v)
	if err == io.EOF {
		return nil
	}
	return err
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
		jobEntry, ok := value.(JobStoreEntry)
		if ok {
			if jobEntry.JobType == CasaBackupResource {
				jobEntry.UpdateTimestamp = timestamp
				m[key] = jobEntry
				updatedTimestamp = true
				// We expect there to be only 1 K8S_SNAP job
				// But, for whateever reason, there are more than 1,
				// we update the timestamps of all the backup jobs.
				// Hence, we do not "break" here
			}
		}
	}
	if updatedTimestamp {
		err := vs.saveJobsStore(&m)
		if err != nil {
			vs.log.Error(err, "Failed to load the Jobs Store")
			return err
		}
	}

	return nil
}
