package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func main() {
	config := getRESTConfig()
	_ = kubernetesClient(config)
	_ = kubeBuilderClient(config)
	//_ = testCreateOrPatch(config)
	//_ = testPatchStatus(config)
}

func getRESTConfig() *rest.Config {
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
