package do

import (
	"context"
	"strings"

	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/viveksinghggits/kluster/pkg/apis/viveksingh.dev/v1alpha1"
)

func Create(c kubernetes.Interface, spec v1alpha1.KlusterSpec) (string, error) {
	token, err := getToken(c, spec.TokenSecret)
	if err != nil {
		return "", err
	}

	client := godo.NewFromToken(token)

	request := &godo.KubernetesClusterCreateRequest{
		Name:        spec.Name,
		RegionSlug:  spec.Region,
		VersionSlug: spec.Version,
		NodePools: []*godo.KubernetesNodePoolCreateRequest{
			&godo.KubernetesNodePoolCreateRequest{
				Size:  spec.NodePools[0].Size,
				Name:  spec.NodePools[0].Size,
				Count: spec.NodePools[0].Count,
			},
		},
	}

	cluster, _, err := client.Kubernetes.Create(context.Background(), request)
	if err != nil {
		return "", err
	}

	return cluster.ID, nil
}

func getToken(client kubernetes.Interface, sec string) (string, error) {
	namespace := strings.Split(sec, "/")[0]
	name := strings.Split(sec, "/")[1]
	s, err := client.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(s.Data["token"]), nil
}
