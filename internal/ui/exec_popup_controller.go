package ui

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type execPopupModel struct {
	textInput textinput.Model
}

func (p *execPopupModel) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return true
		case tea.KeyEnter:
			// Save the label
			// p.labeler.SetLabel(p.textInput.Value())
			return true
		}
	}

	p.textInput, _ = p.textInput.Update(msg)

	return false
}

func (model *execPopupModel) View(width, height int) string {
	return RenderExecPopup(model, width, height)
}

// ExecInPod executes a command inside a Kubernetes container
func execInPod(clientset kubernetes.Interface, config *rest.Config, namespace, podName, containerName string, command []string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   command,
			Container: containerName,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, runtime.ParameterCodec(scheme.ParameterCodec))

	// Create an executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to initialize executor: %w", err)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func NewExecPopupModel(client conn.KhronosConn, resource types.Resource) Popup {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	// Command to execute inside the container
	command := []string{"sh", "-c", "echo Hello from inside the container"}

	// Execute the command
	output, err := execInPod(client.Client, client.Config, resource.GetNamespace(), resource.GetName(), "etcd", command)
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}

	ti.SetValue(output)

	return &execPopupModel{textInput: ti}
}

func RenderExecPopup(model *execPopupModel, width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(width - 2).
		Height(5).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(fmt.Sprintf(
		"Add a label to this timestamp\n\n%s\n\n%s",
		model.textInput.View(),
		"(esc to quit)",
	))
}
