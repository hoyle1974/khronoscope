package resources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type podLogCollector struct {
	client    kubernetes.Interface
	namespace string
	podName   string
	logs      []string
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
}

func NewPodLogCollector(client kubernetes.Interface, namespace, podName string) *podLogCollector {
	plc := &podLogCollector{
		client:    client,
		namespace: namespace,
		podName:   podName,
		stopCh:    make(chan struct{}),
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
					plc.logs = append(plc.logs, logData)
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

func (plc *podLogCollector) Logs() []string {
	plc.mu.Lock()
	defer plc.mu.Unlock()
	return append([]string{}, plc.logs...) // Return a copy to avoid race conditions
}

type LogCollector struct {
	lock       sync.RWMutex
	client     conn.KhronosConn
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
		go plc.Stop()
		delete(l.collectors, r.GetUID())
	} else {
		l.collectors[r.GetUID()] = NewPodLogCollector(l.client.Client, r.GetNamespace(), r.GetName())
	}
}

func (l *LogCollector) IsLogging(uid string) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	_, ok := l.collectors[uid]
	return ok
}

func (l *LogCollector) GetLogs(uid string) []string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	c, ok := l.collectors[uid]
	if ok {
		return c.Logs()
	}

	return []string{}
}

func NewLogCollector(client conn.KhronosConn) *LogCollector {
	return &LogCollector{
		client:     client,
		collectors: map[string]*podLogCollector{},
	}
}
