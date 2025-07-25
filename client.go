package main

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/clog"
	"github.com/google/uuid"
	"google.golang.org/api/compute/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
)

func New(svc compute.Service, project, zone string) kubernetes.Interface {
	return clientset{svc: svc, project: project, zone: zone}
}

type clientset struct {
	kubernetes.Interface
	svc           compute.Service
	project, zone string
}

func (c clientset) CoreV1() corev1typed.CoreV1Interface {
	return corev1_{svc: c.svc, project: c.project, zone: c.zone}
}

type corev1_ struct {
	corev1typed.CoreV1Interface
	svc           compute.Service
	project, zone string
}

func (c corev1_) Pods(namespace string) corev1typed.PodInterface {
	return &pods{
		svc:     c.svc,
		project: c.project,
		zone:    c.zone,
		suffix:  uuid.New().String()[0:8],
	}
}

type pods struct {
	project, zone string
	svc           compute.Service
	suffix        string
}

func (p *pods) Create(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions) (*corev1.Pod, error) {
	if pod.ObjectMeta.GenerateName != "" {
		pod.ObjectMeta.Name = pod.ObjectMeta.GenerateName + p.suffix
	}
	instance := ToInstance(pod, p.project, p.zone)
	op, err := p.svc.Instances.Insert(p.project, p.zone, &instance).Do()
	if err != nil {
		return nil, fmt.Errorf("error creating instance: %v", err)
	}

	op, err = p.svc.ZoneOperations.Wait(p.project, p.zone, op.Name).Do()
	if err != nil {
		return nil, fmt.Errorf("error waiting for operation: %v", err)
	}
	if op.Error != nil {
		return nil, fmt.Errorf("error creating instance: %v", op.Error)
	}
	return pod, nil
}

func (p *pods) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	op, err := p.svc.Instances.Delete(p.project, p.zone, name).Do()
	if err != nil {
		return fmt.Errorf("error deleting instance: %v", err)
	}
	op, err = p.svc.ZoneOperations.Wait(p.project, p.zone, op.Name).Do()
	if err != nil {
		return fmt.Errorf("error waiting for operation: %v", err)
	}
	if op.Error != nil {
		return fmt.Errorf("error deleting instance: %v", op.Error)
	}
	return nil
}

func (p *pods) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error) {
	log := clog.FromContext(ctx)
	instance, err := p.svc.Instances.Get(p.project, p.zone, name).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting instance: %v", err)
	}

	// Try to get pod status from kubelet
	var externalIP string
	if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].AccessConfigs) > 0 {
		externalIP = instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
	}

	if externalIP != "" {
		kubelet := newKubeletClient(externalIP)
		if pod, err := kubelet.GetPod(name); err == nil {
			log.Debugf("got pod from kubelet for %s", name)
			return pod, nil
		} else {
			log.Debugf("kubelet not ready for %s, using instance status: %v", name, err)
		}
	}

	return ToPod(instance), nil
}

func (p *pods) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	log := clog.FromContext(ctx)
	instances, err := p.svc.Instances.List(p.project, p.zone).Do()
	if err != nil {
		return nil, fmt.Errorf("error listing instances: %v", err)
	}

	pods := &corev1.PodList{}
	for _, instance := range instances.Items {
		// Try to get pod status from kubelet
		var externalIP string
		if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].AccessConfigs) > 0 {
			externalIP = instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
		}

		if externalIP != "" {
			kubelet := newKubeletClient(externalIP)
			if pod, err := kubelet.GetPod(instance.Name); err == nil {
				log.Debugf("got pod from kubelet for %s", instance.Name)
				pods.Items = append(pods.Items, *pod)
				continue
			} else {
				log.Debugf("kubelet not ready for %s, using instance status: %v", instance.Name, err)
			}
		}

		// Fall back to instance status
		pods.Items = append(pods.Items, *ToPod(instance))
	}
	return pods, nil
}
