/*
Copyright 2024.

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
	"gomodules.xyz/jsonpatch/v2"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	cu "kmodules.xyz/client-go/client"
	"kmodules.xyz/client-go/client/duck"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	corev1alpha1 "github.com/ArnobKumarSaha/k8s/api/v1alpha1"
)

// MyPodReconciler reconciles a MyPod object
type MyPodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.duck.dev,resources=mypods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.duck.dev,resources=mypods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.duck.dev,resources=mypods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyPod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *MyPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	//klog.Infof("Reconciling %v \n", req.NamespacedName)
	var mypod corev1alpha1.MyPod
	if err := r.Get(ctx, req.NamespacedName, &mypod); err != nil {
		log.Error(err, "unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sel, err := metav1.LabelSelectorAsSelector(mypod.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	var pods corev1.PodList
	err = r.List(context.TODO(), &pods,
		client.InNamespace(mypod.Namespace),
		client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		return ctrl.Result{}, err
	}

	if req.Namespace == "kube-system" && req.Name == "coredns" {
		r.tryPatch(&mypod)
		klog.Infoln("-------------")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *MyPodReconciler) tryPatch(mp *corev1alpha1.MyPod) {
	_, err := cu.CreateOrPatch(context.TODO(), r.Client, mp, func(object client.Object, createOp bool) client.Object {
		in := object.(*corev1alpha1.MyPod)
		controllerutil.AddFinalizer(in, "kubedb.com/finalizer")
		return in
	})
	klog.Errorf("with conversion : %v \n", err)

	_, err = cu.CreateOrPatch(context.TODO(), r.Client, mp, func(object client.Object, createOp bool) client.Object {
		controllerutil.AddFinalizer(object, "kubedb.com/finalizer")
		//fmt.Printf("*********** %+v", object)
		return object
	})
	klog.Errorf("direct approach : %v \n", err)

	dep := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mp.Name,
			Namespace: mp.Namespace,
		},
	}
	_, err = cu.CreateOrPatch(context.TODO(), r.Client, dep, func(object client.Object, createOp bool) client.Object {
		controllerutil.AddFinalizer(object, "kubedb.com/finalizer")
		return object
	})
	klog.Errorf("Try Deployment : %v \n", err)

	transform := func(obj client.Object, createOp bool) client.Object {
		controllerutil.AddFinalizer(obj, "kubedb.com/finalizer")
		return obj
	}

	patch := client.MergeFrom(mp)
	mod := transform(mp.DeepCopyObject().(client.Object), false)

	data, err := patch.Data(mod)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	curJson, _ := json.Marshal(mp)
	modJson, _ := json.Marshal(mod)
	d2, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		panic(err)
	}
	d2Json, _ := json.Marshal(d2)
	fmt.Println(string(d2Json))

	// Try patching
	time.Sleep(time.Second * 2)
	err = r.Client.Patch(context.TODO(), mod, patch)
	if err != nil {
		panic(err)
	}
}

func (r *MyPodReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyPodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return duck.ControllerManagedBy(mgr).
		For(&corev1alpha1.MyPod{}).
		WithUnderlyingTypes(
			ObjectOf(apps.SchemeGroupVersion.WithKind("Deployment")),
			//ObjectOf(apps.SchemeGroupVersion.WithKind("StatefulSet")),
			//ObjectOf(apps.SchemeGroupVersion.WithKind("DaemonSet")),
		).
		Complete(func() duck.Reconciler {
			return new(MyPodReconciler)
		})
}

func ObjectOf(gvk schema.GroupVersionKind) client.Object {
	var u corev1alpha1.MyPod
	u.GetObjectKind().SetGroupVersionKind(gvk)
	return &u
}
