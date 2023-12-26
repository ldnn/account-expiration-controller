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

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	v1alpha2 "kubesphere.io/api/iam/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=user.ks.cloud.cmft,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=user.ks.cloud.cmft,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=user.ks.cloud.cmft,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	user := &v1alpha2.User{}

	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	uState := user.Status.State
	uName := user.Name
	uLastLoginTime := user.Status.LastLoginTime
	uLastTransitionTime := user.Status.LastTransitionTime
	uCreationTimestamp := user.CreationTimestamp.Time
	if uState == "Active" && uName != "admin" {
		switch {
		case uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
			if uCreationTimestamp.AddDate(0, 3, 0).Before(time.Now()) {
				user.Status.State = "Disabled"
				if err := r.Update(ctx, user); err != nil {
					return ctrl.Result{}, err
				}
			}
		case !uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
			switch {
			case uLastTransitionTime.AddDate(0, 3, 0).Before(time.Now()) && uCreationTimestamp.Equal(uLastTransitionTime.Time):
				user.Status.State = "Disabled"
				if err := r.Update(ctx, user); err != nil {
					return ctrl.Result{}, err

				}
			case uLastTransitionTime.AddDate(0, 0, 7).Before(time.Now()) && !uCreationTimestamp.Equal(uLastTransitionTime.Time):
				if uCreationTimestamp.AddDate(0, 3, 0).Before(time.Now()) {
					user.Status.State = "Disabled"
					if err := r.Update(ctx, user); err != nil {
						return ctrl.Result{}, err
					}
				}
			}
		case uLastTransitionTime.IsZero() && !uLastLoginTime.IsZero():
			if uLastLoginTime.AddDate(0, 3, 0).Before(time.Now()) {
				user.Status.State = "Disabled"
				if err := r.Update(ctx, user); err != nil {
					return ctrl.Result{}, err
				}
			}
		case uLastLoginTime.AddDate(0, 3, 0).Before(time.Now()) && uLastTransitionTime.AddDate(0, 0, 7).Before(time.Now()):
			user.Status.State = "Disabled"
			if err := r.Update(ctx, user); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		For(&v1alpha2.User{}).
		Owns(&v1alpha2.User{}).
		Complete(r)
}
