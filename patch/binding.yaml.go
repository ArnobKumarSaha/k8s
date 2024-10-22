package main

import (
	"context"
	"fmt"
	bapi "go.bytebuilders.dev/catalog/api/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
	"time"
)

var sdb = `apiVersion: catalog.appscode.com/v1alpha1
kind: SinglestoreBinding
metadata:
  name: ssbinding
  namespace: demo
spec:
  sourceRef:
    name: sdb-sample
    namespace: demo`

func main() {
	var cur bapi.SinglestoreBinding
	err := yaml.Unmarshal([]byte(sdb), &cur)
	if err != nil {
		panic(err)
	}

	cfg := ctrl.GetConfigOrDie()
	kc, err := client.New(cfg, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		panic(err)
	}

	var genBindObj bapi.GenericBinding
	if err := kc.Get(context.TODO(), types.NamespacedName{
		Namespace: "demo",
		Name:      "ssbinding",
	}, &genBindObj); err != nil {
		panic(err)
	}

	transform := func(obj client.Object, createOp bool) client.Object {
		//in := obj.(*bapi.SinglestoreBinding)
		controllerutil.AddFinalizer(obj, bapi.GetFinalizer())
		return obj
	}

	patch := client.MergeFrom(&cur)
	mod := transform(cur.DeepCopyObject().(client.Object), false)

	data, err := patch.Data(mod)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	curJson, _ := json.Marshal(cur)
	modJson, _ := json.Marshal(mod)
	d2, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		panic(err)
	}
	d2Json, _ := json.Marshal(d2)
	fmt.Println(string(d2Json))

	// Try patching
	time.Sleep(time.Second * 2)
	err = kc.Patch(context.TODO(), mod, patch)
	if err != nil {
		panic(err)
	}
}
