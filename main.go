package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	nJobs := flag.Int("n", 100, "number of jobs to create every period")
	period := flag.Duration("p", 1*time.Minute, "create n jobs every this amount of time")
	image := flag.String("i", "busybox:latest", "image to run in the jobs")
	sleep := flag.Int("s", 10, "number of seconds for which each job should sleep before terminating")
	ttl := flag.Int("ttl", 10, "number of seconds before completed pods are deleted")
	kubeconfig := flag.String("kubeconfig", filepath.Join(os.ExpandEnv("$HOME"), ".kube", "config"), "absolute path to the kubeconfig file, if running outside a cluster")
	flag.Parse()

	ttl32 := int32(*ttl)

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Error getting in-cluster config: %v", err)
	}

	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Printf("Error reading kubeconfig: %v", err)
		}
	}

	if config == nil {
		log.Fatalf("Couldn't connect to cluster")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building k8s client: %v", err)
	}

	jobsPerSec := float64(*nJobs) / period.Seconds()
	timeToBanish := float64(*sleep + *ttl)
	log.Printf("Expected steady state for this configuration: %f pods", jobsPerSec*timeToBanish)

	for {
		time.Sleep(*period / time.Duration(*nJobs))

		name := fmt.Sprintf("jobspam-%d", time.Now().UTC().UnixMilli())
		_, err := clientset.BatchV1().Jobs(metav1.NamespaceDefault).Create(context.TODO(), &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: &ttl32,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyOnFailure,
						Containers: []corev1.Container{
							{
								Name:            "spam",
								Image:           *image,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command: []string{
									"/bin/sh",
									"-c",
								},
								Args: []string{fmt.Sprintf("sleep %d", *sleep)},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("creating job %s: %v", name, err)
		}
	}
}
