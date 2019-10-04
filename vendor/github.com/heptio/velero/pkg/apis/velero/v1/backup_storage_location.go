/*
Copyright 2018 the Velero contributors.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupStorageLocation is a location where Velero stores backup objects.
type BackupStorageLocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BackupStorageLocationSpec   `json:"spec"`
	Status BackupStorageLocationStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BackupStorageLocationList is a list of BackupStorageLocations.
type BackupStorageLocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BackupStorageLocation `json:"items"`
}

// StorageType represents the type of storage that a backup location uses.
// ObjectStorage must be non-nil, since it is currently the only supported StorageType.
type StorageType struct {
	ObjectStorage *ObjectStorageLocation `json:"objectStorage,omitempty"`
}

// ObjectStorageLocation specifies the settings necessary to connect to a provider's object storage.
type ObjectStorageLocation struct {
	// Bucket is the bucket to use for object storage.
	Bucket string `json:"bucket"`

	// Prefix is the path inside a bucket to use for Velero storage. Optional.
	Prefix string `json:"prefix"`
}

// BackupStorageLocationSpec defines the specification for a Velero BackupStorageLocation.
type BackupStorageLocationSpec struct {
	// Provider is the provider of the backup storage.
	Provider string `json:"provider"`

	// Config is for provider-specific configuration fields.
	Config map[string]string `json:"config"`

	StorageType `json:",inline"`

	// AccessMode defines the permissions for the backup storage location.
	AccessMode BackupStorageLocationAccessMode `json:"accessMode,omitempty"`
}

// BackupStorageLocationPhase is the lifecyle phase of a Velero BackupStorageLocation.
type BackupStorageLocationPhase string

const (
	// BackupStorageLocationPhaseAvailable means the location is available to read and write from.
	BackupStorageLocationPhaseAvailable BackupStorageLocationPhase = "Available"

	// BackupStorageLocationPhaseUnavailable means the location is unavailable to read and write from.
	BackupStorageLocationPhaseUnavailable BackupStorageLocationPhase = "Unavailable"
)

// BackupStorageLocationAccessMode represents the permissions for a BackupStorageLocation.
type BackupStorageLocationAccessMode string

const (
	// BackupStorageLocationAccessModeReadOnly represents read-only access to a BackupStorageLocation.
	BackupStorageLocationAccessModeReadOnly BackupStorageLocationAccessMode = "ReadOnly"

	// BackupStorageLocationAccessModeReadWrite represents read and write access to a BackupStorageLocation.
	BackupStorageLocationAccessModeReadWrite BackupStorageLocationAccessMode = "ReadWrite"
)

// TODO(2.0): remove the AccessMode field from BackupStorageLocationStatus.

// BackupStorageLocationStatus describes the current status of a Velero BackupStorageLocation.
type BackupStorageLocationStatus struct {
	Phase              BackupStorageLocationPhase `json:"phase,omitempty"`
	LastSyncedRevision types.UID                  `json:"lastSyncedRevision,omitempty"`
	LastSyncedTime     metav1.Time                `json:"lastSyncedTime,omitempty"`

	// AccessMode is an unused field.
	//
	// Deprecated: there is now an AccessMode field on the Spec and this field
	// will be removed entirely as of v2.0.
	AccessMode BackupStorageLocationAccessMode `json:"accessMode,omitempty"`
}
