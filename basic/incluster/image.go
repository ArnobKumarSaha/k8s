package main

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"kmodules.xyz/go-containerregistry/authn"
	dboldapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func K8sChainOpts(db *dboldapi.MSSQLServer) *k8schain.Options {
	opts := &k8schain.Options{
		Namespace: db.Namespace,
	}
	if db.Spec.PodTemplate == nil || db.Spec.PodTemplate.Spec.ServiceAccountName == "" {
		opts.ServiceAccountName = db.ServiceAccountName()
	} else {
		opts.ServiceAccountName = db.Spec.PodTemplate.Spec.ServiceAccountName
	}
	if db.Spec.PodTemplate != nil && db.Spec.PodTemplate.Spec.ImagePullSecrets != nil {
		for _, ims := range db.Spec.PodTemplate.Spec.ImagePullSecrets {
			opts.ImagePullSecrets = append(opts.ImagePullSecrets, ims.Name)
		}
	}
	return opts
}

func testImageDigest(cc *kubernetes.Clientset, kc client.Client) error {
	var ms dboldapi.MSSQLServer
	err := kc.Get(context.TODO(), types.NamespacedName{
		Name:      "ms-test",
		Namespace: "demo",
	}, &ms)

	nw := time.Now()
	klog.Infof("Starts -> %v \n", nw)
	image, err := authn.ImageWithDigest(cc, "mcr.microsoft.com/mssql/server:2022-CU14-ubuntu-22.04", K8sChainOpts(&ms))
	if err != nil {
		return err
	}
	klog.Infof("Time taken -> %v, %s\n", time.Since(nw), image)
	return nil
}
