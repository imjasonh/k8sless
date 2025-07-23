package main

import (
	"context"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"
	"google.golang.org/api/compute/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (p *pods) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	w := &watcher{
		svc:     p.svc,
		project: p.project,
		zone:    p.zone,
		name:    strings.TrimPrefix(opts.FieldSelector, "metadata.name="),
		ch:      make(chan watch.Event),
		done:    make(chan struct{}),
	}
	go w.watch(ctx)
	return w, nil
}

type watcher struct {
	svc                 compute.Service
	project, zone, name string
	ch                  chan watch.Event
	done                chan struct{}
	stopped             bool
}

func (w *watcher) Stop() {
	if w.stopped {
		return
	}
	w.stopped = true
	close(w.done)
	// Give the goroutine time to exit cleanly
	time.Sleep(100 * time.Millisecond)
	close(w.ch)
}
func (w *watcher) ResultChan() <-chan watch.Event { return w.ch }

func (w *watcher) watch(ctx context.Context) {
	log := clog.FromContext(ctx)
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()

	var lastPod *corev1.Pod

	for {
		select {
		case <-w.done:
			return
		case <-tick.C:
			log.Infof("checking for updates to %s", w.name)
			instance, err := w.svc.Instances.Get(w.project, w.zone, w.name).Do()
			if err != nil {
				log.Errorf("error getting instance %s: %v", w.name, err)
				select {
				case w.ch <- watch.Event{Type: watch.Error, Object: nil}:
				case <-w.done:
				}
				return
			}

			// Get the external IP
			var externalIP string
			if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].AccessConfigs) > 0 {
				externalIP = instance.NetworkInterfaces[0].AccessConfigs[0].NatIP
			}

			if externalIP == "" {
				log.Debugf("instance %s has no external IP yet", w.name)
				continue
			}

			// Query kubelet for actual pod status
			kubelet := newKubeletClient(externalIP)
			pod, err := kubelet.GetPod(w.name)
			if err != nil {
				// If kubelet isn't ready yet, use instance status
				log.Debugf("kubelet not ready, using instance status: %v", err)
				pod = ToPod(instance)
			} else {
				log.With("pod", pod.Name).Infof("got pod from kubelet: phase=%s", pod.Status.Phase)
			}

			// Only send event if pod changed
			if lastPod == nil || lastPod.Status.Phase != pod.Status.Phase {
				eventType := watch.Modified
				if lastPod == nil {
					eventType = watch.Added
				}

				select {
				case w.ch <- watch.Event{Type: eventType, Object: pod}:
					lastPod = pod
				case <-w.done:
					return
				}
			}
		case <-ctx.Done():
			select {
			case w.ch <- watch.Event{Type: watch.Error, Object: nil}:
			case <-w.done:
			}
			return
		}
	}
}
