package shell

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// CommandOutput represents command execution result
type CommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunCommand executes a shell command and returns output
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

// Session represents an interactive shell session
type Session struct {
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stdin    io.WriteCloser
	stderr   io.ReadCloser
	done     chan struct{}
	isClosed bool
}

// NewSession creates a new interactive shell session
// workDir задаёт рабочую директорию; если пустая — используется текущая
func NewSession(workDir string) (*Session, error) {
	cmd := exec.Command("sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if workDir != "" {
		cmd.Dir = workDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		stdout.Close()
		return nil, err
	}

	// Redirect stderr to stdout to preserve order
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stdin.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	return &Session{
		cmd:      cmd,
		stdout:   stdout,
		stdin:    stdin,
		stderr:   nil, // stderr is redirected to stdout
		done:     done,
		isClosed: false,
	}, nil
}

// Write writes data to the shell's stdin
func (s *Session) Write(data []byte) (int, error) {
	if s.isClosed {
		return 0, os.ErrClosed
	}
	return s.stdin.Write(data)
}

// ReadStdout reads available output from stdout
func (s *Session) ReadStdout() string {
	if s.stdout == nil {
		return ""
	}
	buf := make([]byte, 4096)
	// Use non-blocking read with select
	select {
	case <-time.After(10 * time.Millisecond):
		// Timeout - no data available
		return ""
	default:
		// Try to read without blocking
		n, err := s.stdout.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("shell: ReadStdout error: %v", err)
			return ""
		}
		if n > 0 {
			log.Printf("shell: ReadStdout read %d bytes: %q", n, string(buf[:n]))
			return string(buf[:n])
		}
		return ""
	}
}

// ReadStderr reads available output from stderr
func (s *Session) ReadStderr() string {
	if s.stderr == nil {
		return ""
	}
	buf := make([]byte, 4096)
	// Use non-blocking read with select
	select {
	case <-time.After(10 * time.Millisecond):
		// Timeout - no data available
		return ""
	default:
		// Try to read without blocking
		n, err := s.stderr.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("shell: ReadStderr error: %v", err)
			return ""
		}
		if n > 0 {
			log.Printf("shell: ReadStderr read %d bytes: %q", n, string(buf[:n]))
			return string(buf[:n])
		}
		return ""
	}
}

// ClearOutput clears the stdout buffer
func (s *Session) ClearOutput() {
	if s.stdout == nil {
		return
	}
	buf := make([]byte, 4096)
	for {
		n, err := s.stdout.Read(buf)
		if n == 0 || err == io.EOF {
			break
		}
	}
}

// IsRunning checks if the shell session is still running
func (s *Session) IsRunning() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

// Close closes the shell session
func (s *Session) Close() {
	if s.isClosed {
		return
	}
	s.isClosed = true

	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.stderr != nil {
		s.stderr.Close()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
}
