package shell

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type CommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func RunCommand(cmd string, workDir string) (*CommandOutput, error) {
	c := exec.Command("sh", "-c", cmd)
	if workDir != "" {
		c.Dir = workDir
	}
	output, err := c.CombinedOutput()
	if err != nil {
		return &CommandOutput{
			Stdout:   "",
			Stderr:   string(output),
			ExitCode: 1,
		}, err
	}
	return &CommandOutput{
		Stdout:   string(output),
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

type Session struct {
	cmd    *exec.Cmd
	ptty   *os.File
	output chan []byte
	done   chan struct{}
	closed bool
}

func NewSession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	if shellCommand == "" {
		shellCommand = "sh"
	}
	cmd := exec.Command(shellCommand)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if workDir != "" {
		cmd.Dir = workDir
	}

	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	ptty, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}

	pty.Setsize(ptty, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setctty: true,
		Setsid:  true,
	}

	if err := cmd.Start(); err != nil {
		ptty.Close()
		tty.Close()
		return nil, err
	}

	tty.Close()

	output := make(chan []byte, 256)
	done := make(chan struct{})

	session := &Session{
		cmd:    cmd,
		ptty:   ptty,
		output: output,
		done:   done,
	}

	go session.readLoop()

	go func() {
		cmd.Wait()
		close(done)
	}()

	return session, nil
}

func (s *Session) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptty.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("shell: PTY read error: %v", err)
			}
			close(s.output)
			return
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			s.output <- data
		}
	}
}

func (s *Session) Write(data []byte) (int, error) {
	if s.closed {
		return 0, os.ErrClosed
	}
	return s.ptty.Write(data)
}

func (s *Session) Output() <-chan []byte {
	return s.output
}

func (s *Session) Resize(rows, cols int) error {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	ws := &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}
	return pty.Setsize(s.ptty, ws)
}

func (s *Session) ReadStdout() string {
	return ""
}

func (s *Session) ReadStderr() string {
	return ""
}

func (s *Session) ClearOutput() {
}

func (s *Session) IsRunning() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

func (s *Session) Close() {
	if s.closed {
		return
	}
	s.closed = true

	_ = s.ptty.Close()

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
			s.cmd.Process.Kill()
		}
	}
}
