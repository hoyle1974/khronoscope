package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentWatchMe struct {
}

func (n DeploymentWatchMe) Tick() {
}

func (n DeploymentWatchMe) Kind() string {
	return "Deployment"
}

func (n *DeploymentWatchMe) Renderer() ResourceRenderer {
	return nil
}

func (n DeploymentWatchMe) convert(obj runtime.Object) *appsv1.Deployment {
	ret, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	return ret
}

func (n DeploymentWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n DeploymentWatchMe) Add(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(d.ObjectMeta.CreationTimestamp.Time, n.Kind(), d.Namespace, d.Name, d, nil)
}
func (n DeploymentWatchMe) Modified(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), d.Namespace, d.Name, d, nil)

}
func (n DeploymentWatchMe) Del(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), d.Namespace, d.Name, d, nil)
}

func watchForDeployments(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching deployments . . .")
	watchChan, err := k.client.AppsV1().Deployments("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), DeploymentWatchMe{})
}
