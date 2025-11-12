package main

func main() {
	config := GetConfig()
	_ = kubernetesClient(config)

	kc, err := NewClient(config)
	if err != nil {
		panic(err)
	}
	_ = kubeBuilderClient(kc)
	_ = testCreateOrPatch(kc)
	//_ = testPatchStatus(config)
	_ = preferredResourcesWithDiscoveryClient(config)
}
