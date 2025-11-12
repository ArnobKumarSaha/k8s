package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	vsecretapi "go.virtual-secrets.dev/apimachinery/apis/virtual/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scm = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(kubedbscheme.AddToScheme(scm))
	utilruntime.Must(vsecretapi.AddToScheme(scm))
}

func GetConfig() *rest.Config {
	_ = clientgoscheme.AddToScheme(scm)
	_ = dbapi.AddToScheme(scm)
	//_ = kubedbscheme.AddToScheme(scheme)

	ctrl.SetLogger(klog.NewKlogr())
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100
	return cfg
}

func NewClient(cfg *rest.Config) (client.Client, error) {
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, hc)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: scm,
		Mapper: mapper,
	})
}

func kubernetesClient(config *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	return nil
}

func preferredResourcesWithDiscoveryClient(cfg *rest.Config) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}

	preferredResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return err
	}

	klog.Infof("===== Found total %d =====", len(preferredResources))
	for _, p := range preferredResources {
		gv, err := schema.ParseGroupVersion(p.GroupVersion)
		if err != nil {
			continue
		}
		if !strings.Contains(gv.String(), "kubedb.com") {
			continue
		}
		klog.Infof("Found preferred group version: %s", gv)
		for _, res := range p.APIResources {
			klog.Infof("%s", res.Name)
		}
	}
	return nil
}

// See above

func getConfig() *rest.Config {
	//var kubeconfig string
	//if home := homedir.HomeDir(); home != "" {
	//	flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	//} else {
	//	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	//}
	//flag.Parse()

	home := homedir.HomeDir()
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	if err != nil {
		panic(err.Error())
	}
	return config
}

func newClient(config *rest.Config) (client.Client, error) {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	return kc, err
}
