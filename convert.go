package main

import (
	"encoding/json"

	"google.golang.org/api/compute/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ToPod(instance *compute.Instance) *corev1.Pod {
	pod := &corev1.Pod{
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

func ToInstance(pod *corev1.Pod) compute.Instance {
	podb, _ := json.Marshal(pod)
	return compute.Instance{
		Name:        pod.Name,
		MachineType: "projects/jason-chainguard/zones/us-east5-a/machineTypes/c4-standard-4", // TODO: flag

		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{{
				Key:   "podspec",
				Value: &[]string{string(podb)}[0],
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
			Subnetwork: "regions/us-east5/subnetworks/default",
			AccessConfigs: []*compute.AccessConfig{{
				Name: "External NAT",
				Type: "ONE_TO_ONE_NAT",
			}},
		}},
		Tags: &compute.Tags{
			Items: []string{"build"},
		},
		Scheduling: &compute.Scheduling{
			InstanceTerminationAction: "DELETE",
			OnHostMaintenance:         "TERMINATE",
			MaxRunDuration:            &compute.Duration{Seconds: 4 * 60 * 60}, // 4 hours
		},
	}
}
