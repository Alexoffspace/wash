package shell

import (
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func DefaultShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "sh"
}

type CommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func RunCommand(cmd string, workDir string) (*CommandOutput, error) {
	c := exec.Command(DefaultShell(), "-c", cmd)
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
	ptty   io.ReadWriteCloser
	output chan []byte
	done   chan struct{}
	closed bool
}

func (s *Session) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptty.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("shell: session read error: %v", err)
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
