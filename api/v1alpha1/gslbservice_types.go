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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GSLBServiceSpec defines the desired state of GSLBService
type GSLBServiceSpec struct {
	// member describes this cluster's contribution to the global service.
	// +required
	Member GSLBMember `json:"member"`

	// global describes GSLB settings for the service.
	// +required
	Global GSLBGlobal `json:"global"`
}

// GSLBMember describes the local target and how its health is evaluated.
type GSLBMember struct {
	// targetRef points to the workload this member exposes.
	// +required
	TargetRef TargetReference `json:"targetRef"`

	// healthChecks configures how the member's health is determined.
	// +optional
	HealthChecks *HealthChecks `json:"healthChecks,omitempty"`
}

// TargetReference identifies the object backing the member.
type TargetReference struct {
	// kind is the type of the referenced object.
	// +kubebuilder:validation:Enum=HTTPRoute;TCPRoute;Service;Ingress
	// +required
	Kind string `json:"kind"`

	// name is the name of the referenced object.
	// +required
	Name string `json:"name"`

	// namespace is the referenced object's namespace; defaults to the resource namespace.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// HealthChecks configures member health evaluation.
type HealthChecks struct {
	// failureThreshold is the number of consecutive failures before the member is marked unhealthy.
	// +kubebuilder:validation:Minimum=1
	// +optional
	FailureThreshold *int `json:"failureThreshold,omitempty"`

	// lua is an inline Lua health check function.
	// +optional
	Lua *string `json:"lua,omitempty"`

	// luaRef references a Lua health check stored in another object.
	// +optional
	LuaRef *LuaReference `json:"luaRef,omitempty"`

	// kind is the health check type; derived from targetRef or lua when unset.
	// +kubebuilder:validation:Enum=HTTPS;TCP-FULL;TCP-HALF;LUA
	// +optional
	Kind string `json:"kind,omitempty"`

	// path is the URI probed for HTTP-based checks.
	// +optional
	Path *string `json:"path,omitempty"`
}

// LuaReference references a Lua script stored in another object.
type LuaReference struct {
	// targetRef is the kind of object holding the script.
	// +required
	TargetRef string `json:"targetRef"`

	// key is the data key that holds the script.
	// +required
	Key string `json:"key"`
}

// GSLBGlobal describes cluster-wide GSLB settings for the service.
type GSLBGlobal struct {
	// priority orders this cluster among members; lower values win.
	// +kubebuilder:validation:Minimum=1
	// +required
	Priority int32 `json:"priority"`

	// views restricts which network views advertise the service.
	// +kubebuilder:default={hnet}
	// +optional
	// Views []View `json:"views,omitempty"`
}

// View is a network view in which the service is advertised.
// +kubebuilder:validation:Enum=inet;hnet
type View string

const (
	ViewInet View = "inet"
	ViewHnet View = "hnet"
)

// GSLBServiceStatus defines the observed state of GSLBService.
type GSLBServiceStatus struct {
	// member reports the observed state of this cluster's member.
	// +optional
	Member *GSLBMemberStatus `json:"member,omitempty"`

	// conditions represent the current state of the GSLBService resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GSLBMemberStatus reports the observed state of a member.
type GSLBMemberStatus struct {
	// id is the operator-generated GSLB service identifier for this member.
	// +optional
	ID string `json:"id,omitempty"`

	// memberOf is the global service name this member is delegated under
	// +optional
	MemberOf string `json:"memberOf,omitempty"`

	// views is the DNS views the GSLB - service resolves in
	Views []View `json:"views"`

	// healthy indicates whether the member currently passes its health checks.
	// +optional
	Healthy *bool `json:"healthy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Priority",type=integer,JSONPath=".spec.global.priority"
// +kubebuilder:printcolumn:name="Healthy",type=boolean,JSONPath=".status.member.healthy"
// +kubebuilder:printcolumn:name="Accepted",type=string,JSONPath=".status.conditions[?(@.type=='Accepted')].status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Member-Of",type=string,JSONPath=".status.member.memberOf",priority=1
// +kubebuilder:printcolumn:name="Views",type=string,JSONPath=".status.member.views",priority=1

// GSLBService is the Schema for the gslbservices API
type GSLBService struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of GSLBService
	// +required
	Spec GSLBServiceSpec `json:"spec"`

	// status defines the observed state of GSLBService
	// +optional
	Status GSLBServiceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// GSLBServiceList contains a list of GSLBService
type GSLBServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []GSLBService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GSLBService{}, &GSLBServiceList{})
}
