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

package v1beta1

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

	Target         DynamicIngressTarget                 `json:"target"`
	PassiveIngress *DynamicIngressTargetIngressTemplate `json:"passiveIngress,omitempty"`
	ActiveIngress  *DynamicIngressTargetIngressTemplate `json:"activeIngress,omitempty"`
}

// DynamicIngressStatus defines the observed state of DynamicIngress
type DynamicIngressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Value     DynamicIngressStatusValue     `json:"value,omitempty"`
	Condition DynamicIngressStatusCondition `json:"condition,omitempty"`
}

type DynamicIngressStatusValue string

const (
	DynamicIngressNotReady  = DynamicIngressStatusValue("NotReady")
	DynamicIngressHealthy   = DynamicIngressStatusValue("Healthy")
	DynamicIngressUnhealthy = DynamicIngressStatusValue("Unhealthy")
)

type DynamicIngressStatusCondition string

const (
	False = DynamicIngressStatusCondition("False")
	True  = DynamicIngressStatusCondition("True")
	Error = DynamicIngressStatusCondition("Error")
)

type DynamicIngressTargetIngressTemplate struct {
	Metadata DynamicIngressTargetIngressTemplateMetadata `json:"metadata"`
	Spec     networkingv1.IngressSpec                    `json:"spec"`
}

type DynamicIngressTargetIngressTemplateMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type DynamicIngressTarget struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.value"

// DynamicIngress is the Schema for the dynamicingresses API
type DynamicIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicIngressSpec   `json:"spec,omitempty"`
	Status DynamicIngressStatus `json:"status,omitempty"`
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
