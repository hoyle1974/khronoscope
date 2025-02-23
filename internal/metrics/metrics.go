package metrics

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
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

func Log() {
	lock.Lock()
	defer lock.Unlock()
	l := log.Info()
	for k, v := range counter {
		l.Any(k, v)
	}
	l.Msg("Metrics)")
}

func Count(key string, value int) {
	lock.Lock()
	defer lock.Unlock()

	counter[key] += value
}
