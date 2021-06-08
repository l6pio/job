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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var joblog = logf.Log.WithName("job-resource")

func (r *Job) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-l6p-io-v1-job,mutating=true,failurePolicy=fail,sideEffects=None,groups=l6p.io,resources=jobs,verbs=create;update,versions=v1,name=mjob.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Job{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Job) Default() {
	println(">>>>>>>>>>>################")
	joblog.Info("default", "name", r.Name)

	if r.Spec.MinScaleIntervalSeconds == nil {
		r.Spec.MinScaleIntervalSeconds = new(int32)
		*r.Spec.MinScaleIntervalSeconds = 0
	}

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = new(int32)
		*r.Spec.Replicas = 1
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-l6p-io-v1-job,mutating=false,failurePolicy=fail,sideEffects=None,groups=l6p.io,resources=jobs,verbs=create;update,versions=v1,name=vjob.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Job{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Job) ValidateCreate() error {
	joblog.Info("validate create", "name", r.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Job) ValidateUpdate(old runtime.Object) error {
	joblog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Job) ValidateDelete() error {
	joblog.Info("validate delete", "name", r.Name)
	return nil
}
