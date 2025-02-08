package resources

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	"github.com/hoyle1974/khronoscope/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func toPodExtra(r types.Resource) (PodExtra, bool) {
	if r.GetKind() != "Pod" {
		return PodExtra{}, false
	}
	extra := r.GetExtra()
	if extra == nil {
		return PodExtra{}, false
	}

	if extra, ok := extra.(PodExtra); ok { // Type assertion with ok check
		return extra, true
	}

	return PodExtra{}, false
}

func IsLogging(r types.Resource) bool {
	if pe, ok := toPodExtra(r); ok {
		return len(pe.Logging) > 0
	}
	return false
}

func ToggleLogs(r types.Resource, containerName string) {
	if extra, ok := toPodExtra(r); ok {
		if rs, ok := r.(Resource); ok {
			extra = extra.Copy().(PodExtra)

			found := false
			for i, v := range extra.Logging {
				if v == containerName {
					extra.Logging = append(extra.Logging[:i], extra.Logging[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				extra.Logging = append(extra.Logging, containerName)
			}

			rs.Extra = extra
			rs.Timestamp = serializable.Time{Time: time.Now()}

			go _watcher.Update(rs)

			if !found {
				_logCollector.start(r, containerName, func(logs string) {
					// Get the latest resource
					if rs, err := _watcher.data.GetResourceAt(time.Now(), r.GetUID()); err == nil {
						extra = rs.Extra.Copy().(PodExtra)
						extra.Logs = append(extra.Logs, strings.Split(logs, "\n")...)
						rs.Extra = extra
						rs.Timestamp = serializable.Time{Time: time.Now()}
						go _watcher.Update(rs)
					}
				})
			} else {
				_logCollector.stop(r, containerName)
			}
		}

	}
}

type podLogCollector struct {
	client        kubernetes.Interface
	namespace     string
	podName       string
	containerName string
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	onLog         func(logs string)
}

func NewPodLogCollector(l *LogCollector, client kubernetes.Interface, r types.Resource, containerName string, onLog func(logs string)) *podLogCollector {
	plc := &podLogCollector{
		client:        client,
		namespace:     r.GetNamespace(),
		podName:       r.GetName(),
		containerName: containerName,
		stopCh:        make(chan struct{}),
		onLog:         onLog,
	}
	plc.start(l, key(r, containerName))
	return plc
}

func (plc *podLogCollector) getPodLogs(containerName string) error {
	lines := int64(15)
	req := plc.client.CoreV1().Pods(plc.namespace).GetLogs(plc.podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
		TailLines: &lines,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podLogs, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("error opening stream: %w", err)
	}
	defer podLogs.Close()

	reader := bufio.NewReader(podLogs)
	for {
		select {
		case <-plc.stopCh:
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				return fmt.Errorf("log stream closed")
			} else if err != nil {
				return fmt.Errorf("log stream error: %w", err)
			}

			plc.onLog(strings.TrimSpace(line))
		}
	}
}

func (plc *podLogCollector) start(l *LogCollector, key string) {
	plc.wg.Add(1)
	go func() {
		defer plc.wg.Done()
		_ = plc.getPodLogs(plc.containerName)

		// Remove from collectors map when logs stop
		l.lock.Lock()
		delete(l.collectors, key)
		l.lock.Unlock()
	}()
}

func (plc *podLogCollector) Stop() {
	plc.mu.Lock()
	defer plc.mu.Unlock()
	select {
	case <-plc.stopCh:
		return // Already stopped
	default:
		close(plc.stopCh) // Stop goroutine safely
		plc.wg.Wait()     // Wait for it to finish
	}
}

type LogCollector struct {
	lock       sync.RWMutex
	client     conn.KhronosConn
	collectors map[string]*podLogCollector
}

func key(r types.Resource, containerName string) string {
	return r.GetUID() + ":" + r.GetNamespace() + "." + r.GetName() + ":" + containerName
}

func (l *LogCollector) start(r types.Resource, containerName string, onLog func(logs string)) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.collectors[key(r, containerName)] = NewPodLogCollector(l, l.client.Client, r, containerName, onLog)
}

func (l *LogCollector) stop(r types.Resource, containerName string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	k := key(r, containerName)
	plc, ok := l.collectors[k]
	if ok {
		go plc.Stop()
		delete(l.collectors, k)
		return
	}
}

var (
	_logCollector    *LogCollector
	onceLogCollector sync.Once
)

func GetLogCollector(client conn.KhronosConn) *LogCollector {
	onceLogCollector.Do(func() {
		_logCollector = &LogCollector{
			client:     client,
			collectors: map[string]*podLogCollector{},
		}
	})

	return _logCollector
}
