/*
Copyright 2021.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobSpec defines the desired state of Job
type JobSpec struct {
	Template                corev1.PodTemplateSpec `json:"template,omitempty"`
	Replicas                *int32                 `json:"replicas,omitempty"`
	MinScaleIntervalSeconds *int32                 `json:"minScaleIntervalSeconds,omitempty"`
	ActiveSeconds           *int32                 `json:"activeSeconds,omitempty"`
	TTLSecondsAfterFinished *int32                 `json:"ttlSecondsAfterFinished,omitempty"`
	Proxy                   ProxySpec              `json:"proxy,omitempty"`
}

type ProxySpec struct {
	Enabled   bool                        `json:"enabled,omitempty"`
	Port      int32                       `json:"port,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	Kafka     KafkaSpec                   `json:"kafka,omitempty"`
}

type KafkaSpec struct {
	Enabled bool   `json:"enabled,omitempty"`
	Addr    string `json:"addr,omitempty"`
	Topic   string `json:"topic,omitempty"`
}

// JobStatus defines the observed state of Job
type JobStatus struct {
	LabelSelector string      `json:"labelSelector,omitempty"`
	ReadyReplicas int32       `json:"readyReplicas"`
	StartTime     metav1.Time `json:"startTime,omitempty"`
	EndTime       metav1.Time `json:"endTime,omitempty"`
	LastScaleTime metav1.Time `json:"lastScaleTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
//+kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`

// Job is the Schema for the jobs API
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec,omitempty"`
	Status JobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
