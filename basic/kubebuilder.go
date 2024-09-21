package main

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	kmapi "kmodules.xyz/client-go/api/v1"
	cu "kmodules.xyz/client-go/client"
	"kmodules.xyz/client-go/conditions"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scm = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(kubedbscheme.AddToScheme(scm))
}

func kubeBuilderClient(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	var depList appsv1.DeploymentList
	err = kc.List(context.TODO(), &depList)
	if err != nil {
		return err
	}
	klog.Infof("Found %d deployments", len(depList.Items))

	mp := make(map[string]string)
	mp["metadata.name"] = "coredns"
	err = kc.List(context.Background(), &depList, client.MatchingFieldsSelector{Selector: fields.Set(mp).AsSelector()})
	if err != nil {
		return err
	}

	klog.Infof("Found %d deployments with field selector", len(depList.Items))
	return nil
}

func testCreateOrPatch(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	mg := &dbapi.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mg",
			Namespace: "demo",
		},
	}
	klog.Infof("Trying to create mongodb")
	v, err := cu.CreateOrPatch(context.TODO(), kc, mg, func(obj client.Object, createOp bool) client.Object {
		db := obj.(*dbapi.MongoDB)
		db.Spec.Version = "5.0.3"
		db.Spec.Replicas = pointer.Int32(1)
		return db
	})
	if err != nil {
		klog.Infof("%s \n", err.Error())
		return err
	}
	klog.Infof("%+v, %+v", v, mg)
	return nil
}

func testPatchStatus(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	mg := &dbapi.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mg",
			Namespace: "demo",
		},
	}
	klog.Infof("Trying to patching mongodb status")
	v, err := cu.PatchStatus(context.TODO(), kc, mg, func(obj client.Object) client.Object {
		db := obj.(*dbapi.MongoDB)
		db.Status.Conditions = conditions.SetCondition(db.Status.Conditions, kmapi.Condition{
			Type:    "aaaa",
			Status:  "aaa",
			Reason:  "aa",
			Message: "a",
		})
		return db
	})
	if err != nil {
		klog.Infof("%s \n", err.Error())
		return err
	}
	klog.Infof("%+v, %+v =======  %+v \n", v, mg.Spec, mg.Status)
	return nil
}
