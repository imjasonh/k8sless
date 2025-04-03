package main

import (
	"context"
	"flag"

	"github.com/chainguard-dev/clog"
	"google.golang.org/api/compute/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	project = flag.String("project", "jason-chainguard", "GCP project")
	zone    = flag.String("zone", "us-east5-a", "GCP zone")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	log := clog.FromContext(ctx)
	ctx = clog.WithLogger(ctx, log)

	svc, err := compute.NewService(ctx)
	if err != nil {
		log.Fatalf("error creating compute service: %v", err)
	}
	c := New(*svc, *project, *zone)

	// Create a pod
	p, err := c.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test",
				Image: "cgr.dev/chainguard/wolfi-base",
				Args:  []string{"echo", "hello world"},
			}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("error creating pod: %v", err)
	}
	log.Infof("created pod: %s", p.Name)

	// Get the pod
	pod, err := c.CoreV1().Pods("default").Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("error getting pod: %v", err)
	}
	log.Infof("got pod: %s", pod.Name)

	// Watch the pod
	w, err := c.CoreV1().Pods("default").Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + pod.Name,
	})
	defer w.Stop()
	if err != nil {
		log.Fatalf("error creating watcher: %v", err)
	}
	for event := range w.ResultChan() {
		log.Infof("got event: %s", event.Type)

		if event.Type == watch.Error {
			log.Errorf("error watching pod: %v", event.Object)
			break
		}

		ok := false
		pod, ok = event.Object.(*corev1.Pod)
		if !ok {
			log.Errorf("unexpected type: %T", event.Object)
			break
		}
		switch event.Type {
		case watch.Added:
			log.With("pod", pod.Name).Debugf("pod added: %s %s", pod.Name, pod.Status.Phase)
		case watch.Modified:
			log.With("pod", pod.Name).Infof("pod modified: %s %s", pod.Name, pod.Status.Phase)
		case watch.Deleted:
			log.With("pod", pod.Name).Infof("pod deleted: %s %s", pod.Name, pod.Status.Phase)
		}
	}

	// Delete the pod
	if err := c.CoreV1().Pods("default").Delete(ctx, p.Name, metav1.DeleteOptions{}); err != nil {
		log.Fatalf("error deleting pod: %v", err)
	}
	log.Infof("deleted pod: %s", p.Name)
}
