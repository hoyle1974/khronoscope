package metrics

import (
	"fmt"
	"sync"
)

var lock = sync.Mutex{}
var counter = map[string]int{}

func Print() {
	lock.Lock()
	defer lock.Unlock()
	fmt.Println("Metrics:")
	for k, v := range counter {
		fmt.Printf("	%s: %d\n", k, v)
	}
}

func Count(key string, value int) {
	lock.Lock()
	defer lock.Unlock()

	counter[key] += value
}
