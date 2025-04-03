package main

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	restclient "k8s.io/client-go/rest"
)

func (p *pods) Update(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	return nil, errors.New("not implemented")
}
func (p *pods) UpdateStatus(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	return nil, errors.New("not implemented")
}
func (p *pods) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return errors.New("not implemented")
}
func (p *pods) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Pod, err error) {
	return nil, errors.New("not implemented")
}
func (p *pods) Apply(ctx context.Context, pod *applyconfigurationscorev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Pod, err error) {
	return nil, errors.New("not implemented")
}

func (p *pods) ApplyStatus(ctx context.Context, pod *applyconfigurationscorev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Pod, err error) {
	return nil, errors.New("not implemented")
}
func (p *pods) UpdateEphemeralContainers(ctx context.Context, podName string, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	return nil, errors.New("not implemented")
}
func (p *pods) UpdateResize(ctx context.Context, podName string, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	return nil, errors.New("not implemented")
}

func (p *pods) Bind(ctx context.Context, binding *corev1.Binding, opts metav1.CreateOptions) error {
	return errors.New("not implemented")
}

func (p *pods) Evict(ctx context.Context, eviction *policyv1beta1.Eviction) error {
	return errors.New("not implemented")
}

func (p *pods) EvictV1(ctx context.Context, eviction *policyv1.Eviction) error {
	return errors.New("not implemented")
}
func (p *pods) EvictV1beta1(ctx context.Context, eviction *policyv1beta1.Eviction) error {
	return errors.New("not implemented")
}

func (p *pods) ProxyGet(scheme, name, port, path string, params map[string]string) restclient.ResponseWrapper {
	return nil
}

func (p *pods) GetLogs(name string, opts *corev1.PodLogOptions) *restclient.Request { return nil }
