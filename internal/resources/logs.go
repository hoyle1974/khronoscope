package resources

import (
	"sync"

	"github.com/hoyle1974/khronoscope/internal/types"
)

type podLogCollector struct {
}

func NewPodLogCollector(uid string) *podLogCollector {
	return &podLogCollector{}
}

func (plc *podLogCollector) Stop() {

}

type LogCollector struct {
	lock       sync.RWMutex
	collectors map[string]*podLogCollector
}

func (l *LogCollector) ToggleLogs(r types.Resource) {
	if r == nil || r.GetKind() != "Pod" {
		return // Ignore things that are not pods
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	plc, ok := l.collectors[r.GetUID()]
	if ok {
		plc.Stop()
		delete(l.collectors, r.GetUID())
	} else {
		l.collectors[r.GetUID()] = NewPodLogCollector(r.GetUID())
	}
}

func (l *LogCollector) IsLogging(uid string) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	_, ok := l.collectors[uid]
	return ok
}

func NewLogCollector() *LogCollector {
	return &LogCollector{
		collectors: map[string]*podLogCollector{},
	}
}
