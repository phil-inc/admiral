package utils

import (
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClient returns a clientset from inside the cluster
func GetClient() (kubernetes.Interface, error) {
	config, err := GetRestConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Can not create kube client: %v", err)
	}

	return clientset, nil
}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetRestConfig returns a valid rest client for connecting to Kubernetes
func GetRestConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = buildOutOfClusterConfig()
	}
	return config, err
}

// ShrinkStringMap shrinks a map of strings without leaking memory
func ShrinkStringMap(o map[string]string) map[string]string {
	n := make(map[string]string, len(o))

	for k, v := range o {
		n[k] = v
	}

	return n
}

func GenerateUniqueContainerName(pod *v1.Pod, container v1.Container) string {
	return fmt.Sprintf("%s.%s.%s", pod.ObjectMeta.Namespace, pod.Name, container.Name)
}
