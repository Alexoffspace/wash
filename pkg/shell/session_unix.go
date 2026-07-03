//go:build !windows

package shell

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func NewSession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	if shellCommand == "" {
		shellCommand = DefaultShell()
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

func (s *Session) Resize(rows, cols int) error {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	ws := &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}
	return pty.Setsize(s.ptty.(*os.File), ws)
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
