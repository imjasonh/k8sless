package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// kubeletClient queries the kubelet read-only API
type kubeletClient struct {
	baseURL string
	client  *http.Client
}

func newKubeletClient(ip string) *kubeletClient {
	return &kubeletClient{
		baseURL: fmt.Sprintf("http://%s:10255", ip),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// PodList represents the response from kubelet /pods endpoint
type PodList struct {
	Items []corev1.Pod `json:"items"`
}

// GetPods returns all pods from the kubelet
func (k *kubeletClient) GetPods() ([]corev1.Pod, error) {
	resp, err := k.client.Get(k.baseURL + "/pods")
	if err != nil {
		return nil, fmt.Errorf("failed to query kubelet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubelet returned status %d", resp.StatusCode)
	}

	var podList PodList
	if err := json.NewDecoder(resp.Body).Decode(&podList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return podList.Items, nil
}

// GetPod returns a specific pod by name
func (k *kubeletClient) GetPod(name string) (*corev1.Pod, error) {
	pods, err := k.GetPods()
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		if pod.Name == name || pod.Name == name+"-"+name {
			// kubelet may append namespace to pod name
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("pod %s not found", name)
}
