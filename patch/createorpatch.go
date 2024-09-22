package main

import (
	"context"
	"fmt"
	"gomodules.xyz/jsonpatch/v2"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	cu "kmodules.xyz/client-go/client"
	coreutil "kmodules.xyz/client-go/core/v1"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	psapi "kubeops.dev/petset/apis/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"time"
)

var (
	scm = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(kubedbscheme.AddToScheme(scm))
	utilruntime.Must(psapi.AddToScheme(scm))
}

// var ps = `apiVersion: apps/v1
// kind: StatefulSet
var ps = `apiVersion: apps.k8s.appscode.com/v1
kind: PetSet
metadata:
  name: test
  namespace: default
spec:
  selector:
    matchLabels:
      app: nginx
  serviceName: ""
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        imagePullPolicy: IfNotPresent
        name: nginx
        ports:
        - containerPort: 80
          name: web
          protocol: TCP
        resources: {}
  updateStrategy: {}`

func main() {
	//var cur appsv1.StatefulSet
	var cur psapi.PetSet
	err := yaml.Unmarshal([]byte(ps), &cur)
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

	if err := kc.Get(context.TODO(), types.NamespacedName{
		Namespace: cur.GetNamespace(),
		Name:      cur.GetName(),
	}, &cur); err != nil {
		if errors.IsNotFound(err) {
			err = kc.Create(context.TODO(), &cur)
			if err != nil {
				panic(err)
			}
		}
	}

	klog.Infof("First time : generation=%v, rv=%v, port=%v\n", cur.ObjectMeta.Generation, cur.ObjectMeta.ResourceVersion, cur.Spec.Template.Spec.Containers[0].Ports[0])

	transform := func(obj client.Object, createOp bool) client.Object {
		//in := obj.(*appsv1.StatefulSet)
		in := obj.(*psapi.PetSet)

		c := core.Container{
			Name:  "nginx",
			Image: "nginx",
			Ports: []core.ContainerPort{
				{
					Name:          "web",
					ContainerPort: 80,
					//Protocol:      core.ProtocolTCP, // fixes unnecessary patching
				},
			},
			ImagePullPolicy: core.PullIfNotPresent,
		}
		in.Spec.Template.Spec.Containers = coreutil.UpsertContainers(in.Spec.Template.Spec.Containers, []core.Container{c})

		return in
	}

	patch := client.MergeFrom(&cur)
	mod := transform(cur.DeepCopyObject().(client.Object), false)

	data, err := patch.Data(mod)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	// Prints: {"spec":{"template":{"spec":{"containers":[{"image":"nginx","imagePullPolicy":"IfNotPresent","name":"nginx","ports":[{"containerPort":80,"name":"web"}],"resources":{}}]}}}}

	curJson, _ := json.Marshal(cur)
	modJson, _ := json.Marshal(mod)
	d2, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		panic(err)
	}
	d2Json, _ := json.Marshal(d2)
	fmt.Println(string(d2Json))
	// Prints: [{"op":"remove","path":"/spec/template/spec/containers/0/ports/0/protocol"}]

	// Try patching
	time.Sleep(time.Second * 2)
	err = kc.Patch(context.TODO(), mod, patch)
	if err != nil {
		panic(err)
	}

	//var upd appsv1.StatefulSet
	var upd psapi.PetSet
	err = kc.Get(context.TODO(), types.NamespacedName{
		Namespace: cur.GetNamespace(),
		Name:      cur.GetName(),
	}, &upd)
	if err != nil {
		panic(err)
	}
	klog.Infof("2nd time : generation=%v, rv=%v, port=%v\n", upd.ObjectMeta.Generation, upd.ObjectMeta.ResourceVersion, upd.Spec.Template.Spec.Containers[0].Ports[0])

	vt, err := cu.CreateOrPatch(context.TODO(), kc, &upd, transform)
	if err != nil {
		panic(err)
	}
	klog.Infof("%v\n", vt)
	klog.Infof("Last time : generation=%v, rv=%v, port=%v\n", upd.ObjectMeta.Generation, upd.ObjectMeta.ResourceVersion, upd.Spec.Template.Spec.Containers[0].Ports[0])

	// Note that: sts works fine in all cases. It keeps the generation 1.
}
