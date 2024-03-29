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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DynamicIngressStateSpec defines the desired state of DynamicIngressState
type DynamicIngressStateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	FixedResponse *DynamicIngressStateResponse `json:"fixedResponse,omitempty"`
	Probe         *DynamicIngressStateProbe    `json:"probe,omitempty"`
}

type DynamicIngressStateResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type DynamicIngressStateProbe struct {
	Type   ProbeType `json:"type"`
	Method string    `json:"method"`
	Url    string    `json:"url"`
}

// +kubebuilder:validation:Enum=HTTP
type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "HTTP"
)

// DynamicIngressStateStatus defines the observed state of DynamicIngressState
type DynamicIngressStateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Value          DynamicIngressStateStatusValue `json:"value"`
	Response       *DynamicIngressStateResponse   `json:"response"`
	LastUpdateTime metav1.Time                    `json:"lastUpdateTime"`
}

// +kubebuilder:validation:Enum=Healthy;Error
type DynamicIngressStateStatusValue string

const (
	DynamicIngressStateStatusValueHealthy DynamicIngressStateStatusValue = "Healthy"
	DynamicIngressStateStatusValueError   DynamicIngressStateStatusValue = "Error"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=dis
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.value"
//+kubebuilder:printcolumn:name="RESPONSE CODE",type="integer",JSONPath=".status.response.status"
//+kubebuilder:printcolumn:name="LAST UPDATE",type="string",JSONPath=".status.lastUpdateTime"

// DynamicIngressState is the Schema for the dynamicingressstates API
type DynamicIngressState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicIngressStateSpec    `json:"spec,omitempty"`
	Status *DynamicIngressStateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DynamicIngressStateList contains a list of DynamicIngressState
type DynamicIngressStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicIngressState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynamicIngressState{}, &DynamicIngressStateList{})
}
