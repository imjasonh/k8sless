package main

import (
	_ "embed"
	"encoding/json"
	"strings"

	"google.golang.org/api/compute/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed cloud-init.yaml
var cloudInitTemplate string

func ToPod(instance *compute.Instance) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: instance.Name},
		Spec:       podspecFromMetadata(instance),
		Status:     corev1.PodStatus{},
	}

	switch instance.Status {
	case "PROVISIONING", "STAGING":
		pod.Status.Phase = corev1.PodPending

	case "RUNNING":
		pod.Status.Phase = corev1.PodRunning

	case "STOPPED":
		pod.Status.Phase = corev1.PodSucceeded

	case "TERMINATED":
		pod.Status.Phase = corev1.PodFailed

	case "DEPROVISIONING", "REPAIRING", "STOPPING", "SUSPENDED", "SUSPENDING":
		pod.Status.Phase = corev1.PodUnknown

	default:
		pod.Status.Phase = corev1.PodUnknown
	}
	return pod
}

func podspecFromMetadata(instance *compute.Instance) corev1.PodSpec {
	pod := corev1.Pod{}
	if instance.Metadata != nil && instance.Metadata.Items != nil {
		for _, item := range instance.Metadata.Items {
			if item.Key == "podspec" {
				_ = json.Unmarshal([]byte(*item.Value), &pod)
				return pod.Spec
			}
		}
	}
	return corev1.PodSpec{}
}

func ToInstance(pod *corev1.Pod, project, zone string) compute.Instance {
	// Ensure the pod has the required TypeMeta fields for kubelet
	pod.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"}
	podb, _ := json.Marshal(pod)
	return compute.Instance{
		Name:        pod.Name,
		MachineType: "projects/" + project + "/zones/" + zone + "/machineTypes/c4-standard-4", // TODO: flag

		// The cloud-init script in user-data will:
		// 1. Configure kubelet with static pod support
		// 2. Retrieve the pod spec from metadata and save to /etc/kubernetes/manifests/
		// 3. Start kubelet with read-only API on port 10255
		//
		// TODO: Update watcher to query kubelet API for actual pod status
		// TODO: Configure logs and metrics collection to GCP

		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{{
				Key:   "podspec",
				Value: &[]string{string(podb)}[0],
			}, {
				Key:   "user-data",
				Value: &[]string{cloudInitTemplate}[0],
			}, {
				Key:   "google-logging-enabled",
				Value: &[]string{"true"}[0],
			}, {
				Key:   "google-monitoring-enabled",
				Value: &[]string{"true"}[0],
			}},
		},
		Disks: []*compute.AttachedDisk{{
			AutoDelete: true,
			Boot:       true,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: "projects/cos-cloud/global/images/family/cos-stable",
			},
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			Network:    "global/networks/default",
			Subnetwork: "regions/" + regionFromZone(zone) + "/subnetworks/default",
			AccessConfigs: []*compute.AccessConfig{{
				Name: "External NAT",
				Type: "ONE_TO_ONE_NAT",
			}},
		}},
		Tags: &compute.Tags{
			Items: []string{"k8sless", "kubelet-api"},
		},
		Labels: map[string]string{
			"k8sless":   "true",
			"pod-name":  pod.Name,
			"namespace": pod.Namespace,
		},
		ServiceAccounts: []*compute.ServiceAccount{{
			Email: "default",
			Scopes: []string{
				"https://www.googleapis.com/auth/logging.write",
				"https://www.googleapis.com/auth/monitoring.write",
				"https://www.googleapis.com/auth/devstorage.read_only", // For pulling container images
			},
		}},
		Scheduling: &compute.Scheduling{
			InstanceTerminationAction: "DELETE",
			OnHostMaintenance:         "TERMINATE",
			MaxRunDuration:            &compute.Duration{Seconds: 4 * 60 * 60}, // 4 hours
		},
	}
}

func regionFromZone(zone string) string {
	// Zone format is like "us-west4-a", we need "us-west4"
	parts := strings.Split(zone, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return zone
}
