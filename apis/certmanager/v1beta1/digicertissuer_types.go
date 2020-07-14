/*
Copyright 2020 SAP SE.

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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DigicertIssuerSpec defines the desired state of DigicertIssuer
type DigicertIssuerSpec struct {
	// URL is the DigiCert cert-central URL containing the token.
	URL string `json:"url"`

	// Provisioner ...
	Provisioner DigicertProvisioner `json:"provisioner"`
}

type DigicertProvisioner struct {
}

// DigicertIssuerStatus defines the observed state of DigicertIssuer
type DigicertIssuerStatus struct {
	// Conditions ...
	// +optional
	Conditions []DigicertIssuerCondition `json:"conditions,omitempty"`
}

// DigicertIssuerCondition  ...
type DigicertIssuerCondition struct {
	// Type of the condition, currently ('Ready').
	Type ConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}

// ConditionType represents a StepIssuer condition type.
// +kubebuilder:validation:Enum=Ready
type ConditionType string

const (
	// ConditionReady indicates that a StepIssuer is ready for use.
	ConditionReady ConditionType = "Ready"
)

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=True;False;Unknown
type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in
// the condition; "ConditionFalse" means a resource is not in the condition;
// "ConditionUnknown" means kubernetes can't decide if a resource is in the
// condition or not. In the future, we could add other intermediate
// conditions, e.g. ConditionDegraded.
const (
	// ConditionTrue represents the fact that a given condition is true
	ConditionTrue ConditionStatus = "True"

	// ConditionFalse represents the fact that a given condition is false
	ConditionFalse ConditionStatus = "False"

	// ConditionUnknown represents the fact that a given condition is unknown
	ConditionUnknown ConditionStatus = "Unknown"
)

// +kubebuilder:object:root=true

// DigicertIssuer is the Schema for the digicertissuers API
// +kubebuilder:subresource:status
type DigicertIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DigicertIssuerSpec   `json:"spec,omitempty"`
	Status DigicertIssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DigicertIssuerList contains a list of DigicertIssuer
type DigicertIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DigicertIssuer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DigicertIssuer{}, &DigicertIssuerList{})
}
