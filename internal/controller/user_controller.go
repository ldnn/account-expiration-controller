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
	"fmt"
	"os"

	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	v1alpha2 "kubesphere.io/api/iam/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	AppId     string
	AppSecret string
	Api       string
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
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	user := &v1alpha2.User{}

	if err := r.Get(ctx, req.NamespacedName, user); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	uState := user.Status.State
	uName := user.Name
	log.V(1).Info("Check User : ", "username", uName)

	//判断有没有电话标签
	if _, ok := user.Labels["iam.kubesphere.io/origin-uid"]; !ok {
		err := fmt.Errorf("无效用户: \"%v\" ，没有电话号码标签", uName)
		log.V(3).Error(err, "请检查")
		return ctrl.Result{}, err
	}

	uTel := user.Labels["iam.kubesphere.io/origin-uid"]

	//删除在4A中失效的账号
	if !CheckStatus(r.AppId, r.AppSecret, r.Api, uTel) {
		log.V(1).Info("用户对应4A账号失效, delete: ", "username", uName, "tel", uTel, "desc", user.Annotations["kubesphere.io/description"])
		if err := r.Delete(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	//取得账号的三个时间
	uLastLoginTime := user.Status.LastLoginTime
	uLastTransitionTime := user.Status.LastTransitionTime
	uCreationTimestamp := user.CreationTimestamp.Time

	//锁定三个月不登陆的账号
	if uState == "Active" && uName != "admin" {
		if !CheckLock(uLastLoginTime, uLastTransitionTime, uCreationTimestamp) {
			user.Status.State = "Disabled"
			if err := r.Update(ctx, user); err != nil {
				return ctrl.Result{}, err
			}
			log.V(1).Info("User is disabled: ", "username", uName)
		}
	}

	//删除九个月不登陆的账号
	if uState == "Disabled" && uName != "admin" {
		if !CheckDel(uLastLoginTime, uLastTransitionTime, uCreationTimestamp) {
			if err := r.Delete(ctx, user); err != nil {
				return ctrl.Result{}, err
			}
			log.V(1).Info("用户连续九个月不登陆, delete: ", "username", uName, "tel", uTel, "desc", user.Annotations["kubesphere.io/description"])
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {

	log := log.Log.WithName("setup")
	secretPath := "/etc/config"

	// 读取配置
	api, err := os.ReadFile(filepath.Join(secretPath, "api"))
	if err != nil {
		log.Error(err, "failed to read secret file: %v")
		os.Exit(1)
	}
	id, err := os.ReadFile(filepath.Join(secretPath, "id"))
	if err != nil {
		log.Error(err, "failed to read secret file")
		os.Exit(1)
	}

	secret, err := os.ReadFile(filepath.Join(secretPath, "secret"))
	if err != nil {
		log.Error(err, "failed to read secret file")
		os.Exit(1)
	}

	// 打印获取的数据
	fmt.Printf("API: %s\n", api)
	fmt.Printf("ID: %s\n", id)
	fmt.Printf("Secret: %s\n", secret)

	r.AppId = string(id)
	r.AppSecret = string(secret)
	r.Api = string(api)

	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		For(&v1alpha2.User{}).
		Owns(&v1alpha2.User{}).
		Complete(r)
}
