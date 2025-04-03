package main

import (
	"context"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"
	"google.golang.org/api/compute/v1"
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
	}
	go w.watch(ctx)
	return w, nil
}

type watcher struct {
	svc                 compute.Service
	project, zone, name string
	ch                  chan watch.Event
}

func (w watcher) Stop()                          { close(w.ch) }
func (w watcher) ResultChan() <-chan watch.Event { return w.ch }

func (w *watcher) watch(ctx context.Context) {
	log := clog.FromContext(ctx)
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			log.Infof("checking for updates to %s", w.name)
			instance, err := w.svc.Instances.Get(w.project, w.zone, w.name).Do()
			if err != nil {
				log.Errorf("error getting instance %s: %v", w.name, err)
				w.ch <- watch.Event{Type: watch.Error, Object: nil}
				return
			}
			pod := ToPod(instance)
			log.With("pod", pod.Name).Infof("pod status %s", pod.Status.Phase)
			w.ch <- watch.Event{Type: watch.Modified, Object: pod}
		case <-ctx.Done():
			w.ch <- watch.Event{Type: watch.Error, Object: nil}
			return
		}
	}
}
