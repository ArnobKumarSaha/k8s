package main

import (
	"context"

	vsecretapi "go.virtual-secrets.dev/apimachinery/apis/virtual/v1alpha1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	config := GetConfig()
	_ = kubernetesClient(config)

	kc, err := NewClient(config)
	if err != nil {
		panic(err)
	}
	//_ = kubeBuilderClient(kc)
	//_ = testCreateOrPatch(kc)
	//_ = testPatchStatus(config)
	//_ = preferredResourcesWithDiscoveryClient(config)
	err = getTry(kc)
	if err != nil {
		klog.Infof("Error getting try: %v", err)
	}
}

func getTry(kc client.Client) error {
	var vs vsecretapi.Secret
	err := kc.Get(context.TODO(), client.ObjectKey{Name: "vsec", Namespace: "demo"}, &vs)
	if err != nil {
		return err
	}
	klog.Infof("Found virtual secret %s/%s \n", vs.GetNamespace(), vs.GetName())
	klog.Infof("Data=%v StringData=%v SecretStoreName=%v \n", vs.Data, vs.StringData, vs.SecretStoreName)

	for key, value := range vs.Data {
		klog.Infof("Key=%v, Value=%v \n", key, string(value))
	}
	return nil
}
