package main

import (
	"context"
	"fmt"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	cu "kmodules.xyz/client-go/client"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1"
	skapi "kubeops.dev/sidekick/apis/apps/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"time"
)

var sk = `apiVersion: apps.k8s.appscode.com/v1alpha1
kind: Sidekick
metadata:
  annotations:
    meta.helm.sh/release-name: ace
    meta.helm.sh/release-namespace: ace
  creationTimestamp: "2024-10-08T14:55:23Z"
  finalizers:
  - apps.k8s.appscode.com
  generation: 1
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: ace-db
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/name: postgreses.kubedb.com
    archiver: "true"
    helm.sh/chart: ace-v2024.10.7
    helm.toolkit.fluxcd.io/name: ace
    helm.toolkit.fluxcd.io/namespace: kubeops
  name: ace-db-sidekick
  namespace: ace
  ownerReferences:
  - apiVersion: kubedb.com/v1
    blockOwnerDeletion: true
    controller: true
    kind: Postgres
    name: ace-db
    uid: 5848777c-3223-4bc0-a4da-fa382fc56df5
  resourceVersion: "8858"
  uid: 47e5fd55-13a7-4923-a852-2b0746c92fd8
spec:
  containers:
  - args:
    - archive
    env:
    - name: PRIMARY_DNS_NAME
      value: ace-db.ace.svc
    - name: NAMESPACE
      value: ace
    - name: DBNAME
      value: ace-db
    - name: SSL_MODE
      value: disable
    - name: CLIENT_AUTH_MODE
      value: md5
    - name: POSTGRES_USER
      valueFrom:
        secretKeyRef:
          key: username
          name: ace-db-auth
    - name: POSTGRES_PASSWORD
      valueFrom:
        secretKeyRef:
          key: password
          name: ace-db-auth
    - name: AWS_S3_FORCE_PATH_STYLE
      value: "true"
    - name: WALG_S3_PREFIX
      value: s3://backupbucket/ace/ace/backups/ace/ace-db/wal-backup
    - name: AWS_REGION
      value: us-east-1
    - name: AWS_ENDPOINT
      value: https://192.168.0.212:4224
    envFrom:
    - secretRef:
        name: default-storage-cred
    image: ghcr.io/kubedb/postgres-archiver:v0.9.0_15.5-alpine@sha256:771b792e4915dc38bbfcf6a3e9b1ea178ed7ab5aeab85a2a7a8e96535e8efbca
    imagePullPolicy: Always
    name: wal-g
    resources:
      limits:
        memory: 128Mi
      requests:
        memory: 128Mi
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsGroup: 70
      runAsNonRoot: true
      runAsUser: 70
      seccompProfile:
        type: RuntimeDefault
    volumeMounts:
    - mountPath: /var/pv
      name: data
  initContainers:
  - env:
    - name: STANDALONE
      value: "false"
    - name: MAJOR_PG_VERSION
      value: "15"
    - name: SSL
      value: "OFF"
    image: ghcr.io/kubedb/postgres-init:0.15.0@sha256:33a36e2d34f06771160693e88aa5893c358aad3bddbdd0e4df2f746c3d7ae625
    name: postgres-init-container
    resources:
      limits:
        memory: 512Mi
      requests:
        cpu: 200m
        memory: 512Mi
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsGroup: 70
      runAsNonRoot: true
      runAsUser: 70
      seccompProfile:
        type: RuntimeDefault
    volumeMounts:
    - mountPath: /var/pv
      name: data
    - mountPath: /run_scripts
      name: run-scripts
    - mountPath: /scripts
      name: scripts
    - mountPath: /role_scripts
      name: role-scripts
  leader:
    selectionPolicy: First
    selector:
      matchLabels:
        app.kubernetes.io/component: database
        app.kubernetes.io/instance: ace-db
        app.kubernetes.io/managed-by: kubedb.com
        app.kubernetes.io/name: postgreses.kubedb.com
        archiver: "true"
        helm.sh/chart: ace-v2024.10.7
        helm.toolkit.fluxcd.io/name: ace
        helm.toolkit.fluxcd.io/namespace: kubeops
        kubedb.com/role: primary
  restartPolicy: Always
  securityContext:
    fsGroup: 70
    runAsGroup: 70
    runAsUser: 70
  serviceAccountName: ace-db-sidekick`

func __main() {
	var cur skapi.Sidekick
	err := yaml.Unmarshal([]byte(sk), &cur)
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

	var db dbapi.Postgres
	if err := kc.Get(context.TODO(), types.NamespacedName{
		Namespace: "ace",
		Name:      "ace-db",
	}, &db); err != nil {
		if errors.IsNotFound(err) {
			err = kc.Create(context.TODO(), &cur)
			if err != nil {
				panic(err)
			}
		}
	}

	klog.Infof("First time : generation=%v, rv=%v\n", cur.ObjectMeta.Generation, cur.ObjectMeta.ResourceVersion)

	transform := func(obj client.Object, createOp bool) client.Object {
		in := obj.(*skapi.Sidekick)
		//core_util.EnsureOwnerReference(&cur.ObjectMeta, db.AsOwner())
		return in
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

	var upd skapi.Sidekick
	err = kc.Get(context.TODO(), types.NamespacedName{
		Namespace: cur.GetNamespace(),
		Name:      cur.GetName(),
	}, &upd)
	if err != nil {
		panic(err)
	}
	klog.Infof("2nd time : generation=%v, rv=%v\n", upd.ObjectMeta.Generation, upd.ObjectMeta.ResourceVersion)

	vt, err := cu.CreateOrPatch(context.TODO(), kc, &upd, transform)
	if err != nil {
		panic(err)
	}
	klog.Infof("%v\n", vt)
	klog.Infof("Last time : generation=%v, rv=%v\n", upd.ObjectMeta.Generation, upd.ObjectMeta.ResourceVersion)
	test(kc)
}

func test(kc client.Client) {
	one := skapi.Sidekick{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ace-db-sidekick",
			Namespace: "ace",
		},
	}
	vt, err := cu.CreateOrPatch(context.TODO(), kc, &one, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*skapi.Sidekick)
		return in
	})
	if err != nil {
		panic(err)
	}
	klog.Infof("%v %v %v %v \n", vt, one.GetGeneration(), one.GetResourceVersion(), one.Spec.Containers[0].Image)

	two := &skapi.Sidekick{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ace-db-sidekick",
			Namespace: "ace",
		},
	}
	vt, err = cu.CreateOrPatch(context.TODO(), kc, two, func(obj client.Object, createOp bool) client.Object {
		in := obj.(*skapi.Sidekick)
		return in
	})
	if err != nil {
		panic(err)
	}
	klog.Infof("%v %v %v %v \n", vt, two.GetGeneration(), two.GetResourceVersion(), two.Spec.Containers[0].Image)
}
