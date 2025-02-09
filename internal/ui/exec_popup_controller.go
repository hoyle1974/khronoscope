package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

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
	stdin     *io.PipeWriter
	output    string
	errMsg    string
}

func (p *execPopupModel) Init() tea.Cmd {
	return nil
}

func (p *execPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return p, tea.Quit
		case tea.KeyEnter:
			// Send command to the remote shell
			cmd := p.textInput.Value() + "\n"
			_, err := p.stdin.Write([]byte(cmd))
			if err != nil {
				p.errMsg = fmt.Sprintf("Error sending input: %v", err)
			}
			p.textInput.SetValue("") // Clear input
		}
	}

	var cmd tea.Cmd
	p.textInput, cmd = p.textInput.Update(msg)
	return p, cmd
}

func (p *execPopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(50).
		Height(10).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	if p.errMsg != "" {
		return style.Render(fmt.Sprintf("%v", p.errMsg))
	}

	return style.Render(fmt.Sprintf(
		"Remote Shell\n\n%s\n\n> %s\n\n%s",
		p.output,
		p.textInput.View(),
		"(esc to quit)",
	))
}

func execInPod(clientset kubernetes.Interface, config *rest.Config, namespace, podName, containerName string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   command,
			Container: containerName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, runtime.ParameterCodec(scheme.ParameterCodec))

	// Create an executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to initialize executor: %w", err)
	}

	// Stream with interactive supportP
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    true, // Enables interactive session
	})
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

func NewExecPopupModel(client conn.KhronosConn, resource types.Resource, containerName string) Popup {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	// Create pipes for stdin
	stdinReader, stdinWriter := io.Pipe()
	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)

	model := &execPopupModel{
		textInput: ti,
		stdin:     stdinWriter,
	}

	// Run the shell session in a goroutine
	go func() {
		err := execInPod(client.Client, client.Config, resource.GetNamespace(), resource.GetName(), containerName, []string{"sh"}, stdinReader, stdoutBuffer, stderrBuffer)
		if err != nil {
			model.errMsg = fmt.Sprintf("Error: %v", err)
		}
	}()

	// Continuously read output
	go func() {
		for {
			time.Sleep(100 * time.Millisecond) // Polling interval
			model.output = stdoutBuffer.String() + stderrBuffer.String()
		}
	}()

	return model
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

	if model.errMsg != "" {
		return style.Render(fmt.Sprintf("%v", model.errMsg))
	}

	return style.Render(fmt.Sprintf(
		"Add a label to this timestamp\n\n%s\n\n%s",
		model.textInput.View(),
		"(esc to quit)",
	))
}
