package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type execPopupModel struct {
	// textInput textinput.Model
	stdin         *io.PipeWriter
	output        string
	errMsg        string
	cursorVisible bool
	width         int
	height        int
	program       *tea.Program
}

func (p *execPopupModel) OnResize(width, height int) {
	p.width = width
	p.height = height
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (p *execPopupModel) Init() tea.Cmd {
	// return nil
	return doTick()
}

func (p *execPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return p, Close
		case tea.KeyEnter:
			_, _ = p.stdin.Write([]byte("\n")) // Send actual newline
		case tea.KeyBackspace:
			_, _ = p.stdin.Write([]byte("\b")) // Send actual backspace
		case tea.KeyTab:
			_, _ = p.stdin.Write([]byte("\t")) // Send tab character
		case tea.KeySpace:
			_, _ = p.stdin.Write([]byte(" ")) // Send space character
		case tea.KeyEscape:
			_, _ = p.stdin.Write([]byte("\x1b")) // Send escape character
		default:
			if len(msg.Runes) > 0 {
				_, _ = p.stdin.Write([]byte(string(msg.Runes))) // Send normal characters
			}
		}
	case TickMsg:
		p.cursorVisible = !p.cursorVisible
		return p, doTick()
	}

	return p, nil
}

func (p *execPopupModel) Close() {
	p.program.Send(PopupClose{})
}

func (p *execPopupModel) View() string {
	//b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		//BorderStyle(b).
		Padding(0).
		Width(p.width).
		Height(p.height).
		AlignHorizontal(lipgloss.Left).
		AlignVertical(lipgloss.Top)

	if p.errMsg != "" {
		return style.Render(fmt.Sprintf("%v", p.errMsg))
	}

	// Append blinking cursor
	cursor := " "
	if p.cursorVisible {
		cursor = "█"
	}

	p.output = truncateOutput(p.output)

	return style.Render(p.errMsg + "\n" + p.output + cursor)
}

// Truncate output when terminal is cleared and keep only last 100 lines
func truncateOutput(output string) string {
	// Regex to detect terminal clear sequences
	re := regexp.MustCompile(`\x1b\[[0-9;]*[HJ]|\x1b\[2J|\x1b\[H|\x1bc`)

	// Find the last occurrence of a clear sequence
	loc := re.FindStringIndex(output)
	if loc != nil {
		// Truncate everything before and including the clear sequence
		output = output[loc[1]:]
	}

	// Split into lines
	lines := strings.Split(output, "\n")

	// Keep only the last 100 lines if there are more
	if len(lines) > 100 {
		lines = lines[len(lines)-100:]
	}

	// Rejoin the truncated output
	return strings.Join(lines, "\n")
}

func execInPod(clientset kubernetes.Interface, config *rest.Config, namespace, podName, containerName string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	coreclient, err := corev1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	req := coreclient.RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"/bin/sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	// Create an executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to initialize executor: %w", err)
	}

	// Stream with interactive supportP
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
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

func NewExecPopupModel(client conn.KhronosConn, resource types.Resource, containerName string, program *tea.Program) Popup {

	// Create pipes for stdin
	stdinReader, stdinWriter := io.Pipe()
	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)

	model := &execPopupModel{
		stdin:   stdinWriter,
		program: program,
	}

	// Run the shell session in a goroutine
	go func() {
		err := execInPod(client.Client, client.Config, resource.GetNamespace(), resource.GetName(), containerName, stdinReader, stdoutBuffer, stderrBuffer)
		if err != nil {
			model.errMsg = fmt.Sprintf("Error: %v", err)
		}
		model.Close()
	}()

	// Continuously read output
	go func() {
		for {
			time.Sleep(10 * time.Millisecond) // Polling interval
			model.output = stdoutBuffer.String() + stderrBuffer.String()
		}
	}()

	return model
}
