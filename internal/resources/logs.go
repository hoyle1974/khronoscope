package resources

import (
	"bytes"
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
		return pe.Logging
	}
	return false
}

func ToggleLogs(r types.Resource) {
	if extra, ok := toPodExtra(r); ok {
		if rs, ok := r.(Resource); ok {
			extra = extra.Copy().(PodExtra)
			extra.Logging = !extra.Logging
			if !extra.Logging {
				extra.Logs = []string{}
			}
			rs.Extra = extra
			rs.Timestamp = serializable.Time{Time: time.Now()}

			go _watcher.Update(rs)

			if extra.Logging {
				_logCollector.start(r, func(logs string) {
					// Get the latest resource
					for _, rs := range _watcher.data.GetResourcesAt(time.Now(), "Pod", r.GetNamespace()) {
						if rs.GetUID() == r.GetUID() {
							extra = rs.Extra.Copy().(PodExtra)
							extra.Logs = append(extra.Logs, strings.Split(logs, "\n")...)
							rs.Extra = extra
							rs.Timestamp = serializable.Time{Time: time.Now()}
							go _watcher.Update(rs)
						}
					}
				})
			} else {
				_logCollector.stop(r.GetUID())
			}
		}

	}
}

type podLogCollector struct {
	client    kubernetes.Interface
	namespace string
	podName   string
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	onLog     func(logs string)
}

func NewPodLogCollector(client kubernetes.Interface, namespace, podName string, onLog func(logs string)) *podLogCollector {
	plc := &podLogCollector{
		client:    client,
		namespace: namespace,
		podName:   podName,
		stopCh:    make(chan struct{}),
		onLog:     onLog,
	}
	plc.start()
	return plc
}

func (plc *podLogCollector) getPodLogs() (string, error) {
	lines := int64(15)
	req := plc.client.CoreV1().Pods(plc.namespace).GetLogs(plc.podName, &corev1.PodLogOptions{
		TailLines: &lines,
	})

	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error opening stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error copying logs: %w", err)
	}

	return buf.String(), nil
}

func (plc *podLogCollector) start() {
	plc.wg.Add(1)
	go func() {
		defer plc.wg.Done()
		for {
			select {
			case <-plc.stopCh:
				return
			default:
				logData, err := plc.getPodLogs()
				if err == nil {
					plc.mu.Lock()
					plc.onLog(logData)
					plc.mu.Unlock()
				}
				time.Sleep(5 * time.Second) // Poll logs every 5 seconds
			}
		}
	}()
}

func (plc *podLogCollector) Stop() {
	close(plc.stopCh)
	plc.wg.Wait()
}

type LogCollector struct {
	lock       sync.RWMutex
	client     conn.KhronosConn
	collectors map[string]*podLogCollector
}

func (l *LogCollector) start(r types.Resource, onLog func(logs string)) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.collectors[r.GetUID()] = NewPodLogCollector(l.client.Client, r.GetNamespace(), r.GetName(), onLog)
}

func (l *LogCollector) stop(uid string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	plc, ok := l.collectors[uid]
	if ok {
		go plc.Stop()
		delete(l.collectors, uid)
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
