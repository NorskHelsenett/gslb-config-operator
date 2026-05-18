/*
Copyright 2026.

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
	"net"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GSLBConfigSpec defines the desired state of GSLBConfig
type GSLBConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	ServiceID *string `json:"id,omitempty"`

	// +kubebuilder:validation:Required
	Priority int `json:"priority"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=255
	MemberOf string `json:"memberOf"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=255
	FQDN string `json:"fqdn"`

	// +kubebuilder:validation:Required
	IP net.IP `json:"ip"`

	// +optional
	// +kubebuilder:default=443
	Port *string `json:"port,omitempty"`

	// + optional
	Path *string `json:"path,omitempty"`

	// +optional
	Datacenter *string `json:"datacenter,omitempty"`

	// +optional
	Interval *time.Duration `json:"interval"`

	// +optional
	// +kubebuilder:default=3
	FailureThreshold *int `json:"failureThreshold,omitempty"`

	// +optional
	// +kubebuilder:default=HTTPS
	// +kubebuilder:validation:Enum=HTTPS;HTTP;TCP-FULL;TCP-HALF
	CheckType string `json:"check"`

	// +optional
	Script *string `json:"lua,omitempty"`
}

// GSLBConfigStatus defines the observed state of GSLBConfig.
type GSLBConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the GSLBConfig resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GSLBConfig is the Schema for the gslbconfigs API
type GSLBConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of GSLBConfig
	// +required
	Spec GSLBConfigSpec `json:"spec"`

	// status defines the observed state of GSLBConfig
	// +optional
	Status GSLBConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// GSLBConfigList contains a list of GSLBConfig
type GSLBConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []GSLBConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GSLBConfig{}, &GSLBConfigList{})
}
