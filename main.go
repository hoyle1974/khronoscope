package main

import (
	"encoding/gob"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	gob.Register(Resource{})

	// test := NewTemporalMap()
	// test.Add(time.Now(), "A", "Value1")
	// test.Add(time.Now(), "A", "Value2")
	// test.Add(time.Now(), "A", NewResource("abc", time.Now(), "efg", "default", "asdfasdfsa", &corev1.Pod{}))
	// b := test.ToBytes()

	// test2 := NewTemporalMapFromBytes(b)
	// fmt.Println(test2)

	client, err := NewKhronosConnection()
	if err != nil {
		fmt.Printf("Error creating connection: %v", err)
		return
	}

	//filename := "temp.dat"
	filename := ""

	data := NewDataModel()

	if len(filename) > 0 {
		data = NewDataModelFromFile(filename)
	}
	var watcher = NewK8sWatcher(data)

	if len(filename) == 0 {
		watchForDeployments(watcher, client)
		watchForDaemonSet(watcher, client)
		watchForReplicaSet(watcher, client)
		watchForService(watcher, client)
		watchForNamespaces(watcher, client)
		podWatcher := watchForPods(watcher, client)
		watchForNodes(watcher, client, podWatcher)
	}

	appModel := newModel(watcher, data)
	p := tea.NewProgram(appModel)

	appModel.vcr = NewVCRControl(data, func() {
		p.Send(1)
	})

	watcher.OnChange(func() {
		p.Send(1)
	})

	if _, err := p.Run(); err != nil {
		panic(err)
	}

}
