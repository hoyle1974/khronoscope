package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceWatchMe struct {
}

func (n ServiceWatchMe) Tick() {
}

func (n ServiceWatchMe) Kind() string {
	return "Service"
}

func (n *ServiceWatchMe) Renderer() ResourceRenderer {
	return nil
}

func (n ServiceWatchMe) convert(obj runtime.Object) *corev1.Service {
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n ServiceWatchMe) Add(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(service.ObjectMeta.CreationTimestamp.Time, n.Kind(), service.Namespace, service.Name, service, nil)

}
func (n ServiceWatchMe) Modified(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), service.Namespace, service.Name, service, nil)

}
func (n ServiceWatchMe) Del(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(service.ObjectMeta.DeletionTimestamp.Time, n.Kind(), service.Namespace, service.Name, service, nil)

}

func watchForService(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching service . . .")
	watchChan, err := k.client.CoreV1().Services("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), ServiceWatchMe{})
}
