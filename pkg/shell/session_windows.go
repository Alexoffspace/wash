//go:build windows

package shell

import (
	"io"
	"os"
	"os/exec"
	"time"
)

type pipeSession struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *pipeSession) Read(b []byte) (int, error)  { return p.stdout.Read(b) }
func (p *pipeSession) Write(b []byte) (int, error)  { return p.stdin.Write(b) }
func (p *pipeSession) Close() error                  { p.stdin.Close(); return p.stdout.Close() }

func NewSession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	if shellCommand == "" {
		shellCommand = DefaultShell()
	}
	cmd := exec.Command(shellCommand)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if workDir != "" {
		cmd.Dir = workDir
	}

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		stdinR.Close()
		stdinW.Close()
		return nil, err
	}

	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stdoutW

	if err := cmd.Start(); err != nil {
		stdinR.Close()
		stdinW.Close()
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}

	stdinR.Close()
	stdoutW.Close()

	output := make(chan []byte, 256)
	done := make(chan struct{})

	session := &Session{
		cmd:    cmd,
		ptty:   &pipeSession{stdin: stdinW, stdout: stdoutR},
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

func (s *Session) Resize(rows, cols int) error {
	return nil
}

func (s *Session) Close() {
	if s.closed {
		return
	}
	s.closed = true

	_ = s.ptty.Close()

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
		}
	}
}
