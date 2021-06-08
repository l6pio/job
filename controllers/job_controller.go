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

package controllers

import (
	"context"
	"fmt"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	l6piov1 "l6p.io/job/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=l6p.io,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l6p.io,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=l6p.io,resources=jobs/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=apps,resources=replicasets/status,verbs=get

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("job", req.NamespacedName)

	var job l6piov1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		_ = r.deleteReplicaSets(ctx, req)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !job.Status.EndTime.IsZero() {
		if job.Spec.TTLSecondsAfterFinished != nil {
			lastLiveTime := job.Status.EndTime.Add(time.Duration(*job.Spec.TTLSecondsAfterFinished) * time.Second)
			if lastLiveTime.Before(time.Now()) {
				if err := r.deleteJob(ctx, &job); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{Requeue: true, RequeueAfter: lastLiveTime.Sub(time.Now())}, nil
			}
		}
		return ctrl.Result{}, nil
	}

	replicaSets, err := r.getReplicaSets(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(replicaSets.Items) == 0 {
		if err := r.createReplicaSet(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}

		job.Status.StartTime = metav1.Now()
		if err := r.updateJobStatus(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&job, kcore.EventTypeNormal, "Started", fmt.Sprintf("Started job %s", job.Name))
		return ctrl.Result{}, nil
	}

	var jobEndTime *time.Time
	if job.Spec.ActiveSeconds != nil {
		jobEndTime = new(time.Time)
		*jobEndTime = job.Status.StartTime.Add(time.Duration(*job.Spec.ActiveSeconds) * time.Second)
		if jobEndTime.Before(time.Now()) {
			if err := r.deleteReplicaSets(ctx, req); err != nil {
				return ctrl.Result{}, err
			}

			job.Status.ReadyReplicas = 0
			job.Status.EndTime = metav1.Now()
			if err := r.updateJobStatus(ctx, &job); err != nil {
				return ctrl.Result{}, err
			}

			r.Recorder.Event(&job, kcore.EventTypeNormal, "Finished", fmt.Sprintf("Finished job %s", job.Name))

			if job.Spec.TTLSecondsAfterFinished != nil {
				return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(*job.Spec.TTLSecondsAfterFinished) * time.Second}, nil
			}
		}
	}

	rs := &replicaSets.Items[0]
	scaleTriggerTime := job.Status.LastScaleTime.Add(time.Duration(*job.Spec.MinScaleIntervalSeconds) * time.Second)
	if job.Status.LastScaleTime.IsZero() || scaleTriggerTime.Before(time.Now()) {
		if *job.Spec.Replicas != *rs.Spec.Replicas {
			if *job.Spec.Replicas < *rs.Spec.Replicas {
				rs.Spec.Replicas = int32Ptr(*rs.Spec.Replicas - 1)
				r.Recorder.Event(&job, kcore.EventTypeNormal, "ScaleDown", fmt.Sprintf("Scale down replicas to %d", *rs.Spec.Replicas))
			} else if *job.Spec.Replicas > *rs.Spec.Replicas {
				rs.Spec.Replicas = int32Ptr(*rs.Spec.Replicas + 1)
				r.Recorder.Event(&job, kcore.EventTypeNormal, "ScaleUp", fmt.Sprintf("Scale up replicas to %d", *rs.Spec.Replicas))
			}

			if err := r.updateReplicaSet(ctx, rs); err != nil {
				return ctrl.Result{}, err
			}

			job.Status.ReadyReplicas = rs.Status.ReadyReplicas
			job.Status.LastScaleTime = metav1.Now()
			if err := r.updateJobStatus(ctx, &job); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	if job.Status.ReadyReplicas != rs.Status.ReadyReplicas {
		job.Status.ReadyReplicas = rs.Status.ReadyReplicas
		if err := r.updateJobStatus(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}
	}

	if jobEndTime != nil && jobEndTime.Before(scaleTriggerTime) {
		return ctrl.Result{Requeue: true, RequeueAfter: jobEndTime.Sub(time.Now())}, nil
	} else {
		return ctrl.Result{Requeue: true, RequeueAfter: scaleTriggerTime.Sub(time.Now())}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&l6piov1.Job{}).
		Complete(r)
}

func (r *JobReconciler) updateJobStatus(ctx context.Context, job *l6piov1.Job) error {
	if err := r.Status().Update(ctx, job); err != nil {
		r.Log.Error(err, "unable to update job status")
		return err
	}
	return nil
}

func (r *JobReconciler) deleteJob(ctx context.Context, job *l6piov1.Job) error {
	if err := r.Delete(ctx, job); err != nil {
		r.Log.Error(err, "unable to delete job")
		return err
	}
	return nil
}

func (r *JobReconciler) createReplicaSet(ctx context.Context, job *l6piov1.Job) error {
	jobLabels := map[string]string{
		"l6p-app": job.Name,
	}

	replicaSet := &kapps.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", job.Name),
			Namespace:    job.Namespace,
			Labels:       jobLabels,
		},
		Spec: kapps.ReplicaSetSpec{
			Template: job.Spec.Template,
			Replicas: job.Spec.Replicas,
		},
	}

	replicaSet.Spec.Replicas = int32Ptr(0)
	replicaSet.Spec.Template.Labels = jobLabels
	replicaSet.Spec.Selector = &metav1.LabelSelector{MatchLabels: jobLabels}

	if job.Spec.Proxy.Enabled {
		proxyContainer := kcore.Container{
			Name:            "l6p-proxy",
			Image:           "localhost:32000/proxy:latest",
			Command:         []string{"/proxy"},
			Args:            []string{"--addr", fmt.Sprintf(":%d", job.Spec.Proxy.Port), job.Name},
			Resources:       job.Spec.Proxy.Resources,
			ImagePullPolicy: "Always",
		}

		containers := replicaSet.Spec.Template.Spec.Containers
		for i := 0; i < len(containers); i++ {
			containers[i].Env = append(containers[i].Env,
				kcore.EnvVar{
					Name:  "http_proxy",
					Value: fmt.Sprintf("http://127.0.0.1:%d", job.Spec.Proxy.Port),
				},
				kcore.EnvVar{
					Name:  "https_proxy",
					Value: fmt.Sprintf("http://127.0.0.1:%d", job.Spec.Proxy.Port),
				},
			)
		}

		if job.Spec.Proxy.Kafka.Enabled {
			proxyContainer.Env = append(proxyContainer.Env,
				kcore.EnvVar{
					Name:  "KAFKA_ADDR",
					Value: job.Spec.Proxy.Kafka.Addr,
				},
				kcore.EnvVar{
					Name:  "KAFKA_TOPIC",
					Value: job.Spec.Proxy.Kafka.Topic,
				},
			)
		}

		replicaSet.Spec.Template.Spec.Containers = append(containers, proxyContainer)
	}

	if err := r.Create(ctx, replicaSet); err != nil {
		r.Log.Error(err, "unable to create replicaset")
		return err
	}
	return nil
}

func (r *JobReconciler) getReplicaSets(ctx context.Context, req ctrl.Request) (*kapps.ReplicaSetList, error) {
	var replicaSets kapps.ReplicaSetList
	labels := client.MatchingLabels{"l6p-app": req.Name}
	if err := r.List(ctx, &replicaSets, client.InNamespace(req.Namespace), labels); err != nil {
		r.Log.Error(err, "unable to list replicasets")
		return nil, err
	}
	return &replicaSets, nil
}

func (r *JobReconciler) deleteReplicaSets(ctx context.Context, req ctrl.Request) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{"l6p-app": req.Name},
	}

	if err := r.DeleteAllOf(ctx, &kapps.ReplicaSet{}, opts...); err != nil {
		r.Log.Error(err, "unable to delete replicaset")
		return err
	}
	return nil
}

func (r *JobReconciler) updateReplicaSet(ctx context.Context, rs *kapps.ReplicaSet) error {
	if err := r.Update(ctx, rs); err != nil {
		r.Log.Error(err, "unable to update replicaset")
		return err
	}
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
