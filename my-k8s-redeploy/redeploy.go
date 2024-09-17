package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Load kubeconfig from default location
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatalf("error building kubeconfig: %v", err)
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("error creating Kubernetes clientset: %v", err)
	}

	// List all Pods in all namespaces
	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("error listing pods: %v", err)
	}

	for _, pod := range pods.Items {
		// Check if the pod name contains "database"
		if containsIgnoreCase(pod.Name, "database") {
			fmt.Printf("Found pod: %s/%s\n", pod.Namespace, pod.Name)
			deploymentName := pod.Labels["app"]
			if deploymentName != "" {
				fmt.Printf("Redeploying associated deployment: %s/%s\n", pod.Namespace, deploymentName)
				err := restartDeployment(clientset, pod.Namespace, deploymentName)
				if err != nil {
					log.Printf("Failed to restart deployment %s/%s: %v", pod.Namespace, deploymentName, err)
				}
			} else {
				log.Printf("Pod %s/%s does not have an associated deployment", pod.Namespace, pod.Name)
			}
		}
	}
}

// containsIgnoreCase checks if a string contains another string, case-insensitive
func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

// restartDeployment performs a graceful restart of the given deployment
func restartDeployment(clientset *kubernetes.Clientset, namespace, name string) error {
	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update the deployment's annotations to trigger a rollout restart
	newDeployment := deployment.DeepCopy()
	if newDeployment.Spec.Template.Annotations == nil {
		newDeployment.Spec.Template.Annotations = make(map[string]string)
	}
	newDeployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Apply the updated Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), newDeployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
