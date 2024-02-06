/*
Copyright 2023.

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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DynamicIngressSpec defines the desired state of DynamicIngress
type DynamicIngressSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Target string `json:"target"`
	// +optional
	PassiveIngress *DynamicIngressTemplate `json:"passiveIngress,omitempty"`
	// +optional
	ActiveIngress    *DynamicIngressTemplate `json:"activeIngress,omitempty"`
	State            string                  `json:"state"`
	SuccessfulStatus int                     `json:"successfulStatus"`
	// +kubebuilder:default=retain
	FailurePolicy    FailurePolicy          `json:"failurePolicy,omitempty"`
	ExpectedResponse DynamicIngressExpected `json:"expectedResponse"`
}

// +kubebuilder:validation:Enum=retain;passive;active
type FailurePolicy string

const (
	FailurePolicyRetain  FailurePolicy = "retain"
	FailurePolicyPassive FailurePolicy = "passive"
	FailurePolicyActive  FailurePolicy = "active"
)

// DynamicIngressStatus defines the observed state of DynamicIngress
type DynamicIngressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Value     DynamicIngressStatusValue     `json:"value,omitempty"`
	Condition DynamicIngressStatusCondition `json:"condition,omitempty"`
}

type DynamicIngressStatusValue string

const (
	DynamicIngressHealthy DynamicIngressStatusValue = "Healthy"
	DynamicIngressError   DynamicIngressStatusValue = "Error"
)

type DynamicIngressStatusCondition string

const (
	DynamicIngressStatusConditionPassive DynamicIngressStatusCondition = "Passive"
	DynamicIngressStatusConditionActive  DynamicIngressStatusCondition = "Active"
	DynamicIngressStatusConditionError   DynamicIngressStatusCondition = "Error"
)

type DynamicIngressTemplate struct {
	Template DynamicIngressTargetIngressTemplate `json:"template"`
}

type DynamicIngressTargetIngressTemplate struct {
	Metadata DynamicIngressTargetIngressTemplateMetadata `json:"metadata"`
	Spec     networkingv1.IngressSpec                    `json:"spec"`
}

type DynamicIngressTargetIngressTemplateMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type DynamicIngressExpected struct {
	Body        string      `json:"body"`
	CompareType CompareType `json:"compareType"`
	// +kubebuilder:default=strict
	ComparePolicy ComparePolicy `json:"comparePolicy"`
}

// +kubebuilder:validation:Enum=plaintext;json
type CompareType string

const (
	CompareTypePlaintext CompareType = "plaintext"
	CompareTypeJson      CompareType = "json"
)

// +kubebuilder:validation:Enum=strict;contains
type ComparePolicy string

const (
	ComparePolicyStrict   ComparePolicy = "strict"
	ComparePolicyContains ComparePolicy = "contains"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,shortName=di
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".spec.state"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.value"
//+kubebuilder:printcolumn:name="CONDITION",type="string",JSONPath=".status.condition"

// DynamicIngress is the Schema for the dynamicingresses API
type DynamicIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicIngressSpec    `json:"spec,omitempty"`
	Status *DynamicIngressStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DynamicIngressList contains a list of DynamicIngress
type DynamicIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynamicIngress{}, &DynamicIngressList{})
}
