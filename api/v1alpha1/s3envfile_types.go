/*
Copyright 2025 Nikolas Sepos.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3EnvFileSpec defines the desired state of S3EnvFile.
type S3EnvFileSpec struct {
	// Bucket is the name of the S3 bucket
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// Key is the name of the S3 object
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// Region is the AWS region of the S3 bucket
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// ConfigMapName is the name of the ConfigMap to create
	ConfigMapName string `json:"configMapName"`
}

// S3EnvFileStatus defines the observed state of S3EnvFile.
type S3EnvFileStatus struct {
	LastModified metav1.Time `json:"lastModified"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// S3EnvFile is the Schema for the s3envfiles API.
type S3EnvFile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   S3EnvFileSpec   `json:"spec,omitempty"`
	Status S3EnvFileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// S3EnvFileList contains a list of S3EnvFile.
type S3EnvFileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []S3EnvFile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&S3EnvFile{}, &S3EnvFileList{})
}
