//go:build windows

package shell

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var kernel32 = windows.NewLazySystemDLL("kernel32.dll")

var (
	procCreatePseudoConsole         = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole         = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole          = kernel32.NewProc("ClosePseudoConsole")
	procGetConsoleOutputCP          = kernel32.NewProc("GetConsoleOutputCP")
	procInitializeProcThreadAttributeList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute   = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList    = kernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
)

type startupInfoEx struct {
	StartupInfo   windows.StartupInfo
	AttributeList uintptr
}

// ---------- ConPTY ----------

type conPty struct {
	hPC          windows.Handle
	hInputWrite  windows.Handle
	hOutputRead  windows.Handle
	attrListBuf  []byte
	procHandle   windows.Handle
}

func newConPty(cols, rows int16) (*conPty, error) {
	var inputR, inputW windows.Handle
	if err := windows.CreatePipe(&inputR, &inputW, nil, 0); err != nil {
		return nil, err
	}
	var outputR, outputW windows.Handle
	if err := windows.CreatePipe(&outputR, &outputW, nil, 0); err != nil {
		windows.CloseHandle(inputR)
		windows.CloseHandle(inputW)
		return nil, err
	}

	var hPC windows.Handle
	ret, _, _ := procCreatePseudoConsole.Call(
		uintptr(uint32(uint16(rows))<<16|uint32(uint16(cols))),
		uintptr(inputR),
		uintptr(outputW),
		0,
		uintptr(unsafe.Pointer(&hPC)),
	)
	if int32(ret) < 0 {
		windows.CloseHandle(inputR)
		windows.CloseHandle(inputW)
		windows.CloseHandle(outputR)
		windows.CloseHandle(outputW)
		return nil, windows.Errno(ret)
	}
	windows.CloseHandle(inputR)
	windows.CloseHandle(outputW)

	return &conPty{
		hPC:         hPC,
		hInputWrite: inputW,
		hOutputRead: outputR,
	}, nil
}

func (c *conPty) Read(b []byte) (int, error) {
	var raw [4096]byte
	var n uint32
	err := windows.ReadFile(c.hOutputRead, raw[:], &n, nil)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}

	u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&raw[0])), n/2)
	runes := utf16.Decode(u16)
	written := 0
	for _, r := range runes {
		written += utf8.EncodeRune(b[written:], r)
	}
	return written, nil
}

func (c *conPty) Write(b []byte) (int, error) {
	// ConPTY input expects UTF-16LE. Normalize LF→CR (Enter key).
	normalized := bytes.ReplaceAll(b, []byte{'\n'}, []byte{'\r'})
	runes := []rune(string(normalized))
	u16 := utf16.Encode(runes)
	u16bytes := unsafe.Slice((*byte)(unsafe.Pointer(&u16[0])), len(u16)*2)

	var n uint32
	err := windows.WriteFile(c.hInputWrite, u16bytes, &n, nil)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *conPty) Close() error {
	procClosePseudoConsole.Call(uintptr(c.hPC))
	if c.procHandle != 0 {
		windows.TerminateProcess(c.procHandle, 1)
		windows.CloseHandle(c.procHandle)
	}
	windows.CloseHandle(c.hInputWrite)
	windows.CloseHandle(c.hOutputRead)
	if len(c.attrListBuf) > 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&c.attrListBuf[0])))
	}
	return nil
}

func (c *conPty) Resize(cols, rows int16) error {
	procResizePseudoConsole.Call(uintptr(c.hPC), uintptr(uint32(uint16(rows))<<16|uint32(uint16(cols))))
	return nil
}

func (c *conPty) startProcess(shellCommand, workDir string) error {
	shellPath, err := exec.LookPath(shellCommand)
	if err != nil {
		return err
	}
	appName, err := windows.UTF16PtrFromString(shellPath)
	if err != nil {
		return err
	}
	cmdLine := `"` + shellPath + `"`
	cmdLinePtr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		return err
	}

	var attrListSize uintptr
	procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrListSize)))

	c.attrListBuf = make([]byte, attrListSize)
	ret, _, _ := procInitializeProcThreadAttributeList.Call(
		uintptr(unsafe.Pointer(&c.attrListBuf[0])),
		1, 0, uintptr(unsafe.Pointer(&attrListSize)),
	)
	if ret == 0 {
		return windows.GetLastError()
	}

	ret, _, _ = procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&c.attrListBuf[0])),
		0, procThreadAttributePseudoConsole,
		uintptr(c.hPC), unsafe.Sizeof(c.hPC), 0, 0,
	)
	if ret == 0 {
		c.cleanupAttrList()
		return windows.GetLastError()
	}

	siEx := startupInfoEx{
		StartupInfo: windows.StartupInfo{Flags: extendedStartupInfoPresent},
		AttributeList: uintptr(unsafe.Pointer(&c.attrListBuf[0])),
	}
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))

	env := buildWindowsEnvStr()

	var workDirPtr *uint16
	if workDir != "" {
		workDirPtr, _ = windows.UTF16PtrFromString(workDir)
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		appName, cmdLinePtr,
		nil, nil, false,
		windows.CREATE_UNICODE_ENVIRONMENT|extendedStartupInfoPresent,
		&env[0], workDirPtr,
		(*windows.StartupInfo)(unsafe.Pointer(&siEx)),
		&pi,
	)
	if err != nil {
		c.cleanupAttrList()
		return err
	}
	windows.CloseHandle(pi.Thread)
	c.procHandle = pi.Process
	return nil
}

func (c *conPty) cleanupAttrList() {
	if len(c.attrListBuf) > 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&c.attrListBuf[0])))
		c.attrListBuf = nil
	}
}

// ---------- Pipe fallback ----------

type pipeSession struct {
	stdin  io.WriteCloser
	stdout io.Reader
}

func (p *pipeSession) Read(b []byte) (int, error) { return p.stdout.Read(b) }

func (p *pipeSession) Write(b []byte) (int, error) {
	return p.stdin.Write(b)
}

func (p *pipeSession) Close() error {
	p.stdin.Close()
	if c, ok := p.stdout.(io.Closer); ok {
		c.Close()
	}
	return nil
}

type lineWriter struct {
	ptty    io.ReadWriteCloser
	output  chan<- []byte
	lineBuf []byte
}

func (w *lineWriter) Read(b []byte) (int, error) { return w.ptty.Read(b) }

func (w *lineWriter) Write(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		ch := b[i]
		switch {
		case ch == 0x7f:
			if len(w.lineBuf) > 0 {
				// remove last rune from lineBuf
				_, size := utf8.DecodeLastRune(w.lineBuf)
				w.lineBuf = w.lineBuf[:len(w.lineBuf)-size]
				for j := 0; j < size; j++ {
					w.output <- []byte{'\b', ' ', '\b'}
				}
			}

		case ch == 0x03:
			w.ptty.Write([]byte{'\n'})
			w.output <- []byte("^C\r\n")
			w.lineBuf = nil

		case ch == '\r' || ch == '\n':
			if len(w.lineBuf) == 0 {
				w.output <- []byte{'\r', '\n'}
				break
			}
			line := make([]byte, len(w.lineBuf)+1)
			copy(line, w.lineBuf)
			line[len(line)-1] = '\n'
			w.ptty.Write(line)
			w.output <- []byte{'\r', '\n'}
			w.lineBuf = nil

		default:
			r, size := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError && size <= 1 {
				w.lineBuf = append(w.lineBuf, ch)
				w.output <- []byte{ch}
				break
			}
			w.lineBuf = append(w.lineBuf, b[i:i+size]...)
			w.output <- b[i : i+size]
			i += size - 1
		}
	}
	return len(b), nil
}

func (w *lineWriter) Close() error { return w.ptty.Close() }

// ---------- Helpers ----------

func getConsoleOutputCP() int {
	ret, _, _ := procGetConsoleOutputCP.Call()
	return int(ret)
}

func getACP() int {
	proc := kernel32.NewProc("GetACP")
	ret, _, _ := proc.Call()
	return int(ret)
}

func getOEMCP() int {
	proc := kernel32.NewProc("GetOEMCP")
	ret, _, _ := proc.Call()
	return int(ret)
}

func decodeConsoleOutput(data []byte) []byte {
	if len(data) == 0 || utf8.Valid(data) {
		return data
	}
	cp := getOEMCP()
	if cp == 65001 || cp == 0 {
		cp = getACP()
	}
	if cp == 65001 || cp == 0 {
		return data
	}
	dec := decoderForCP(cp)
	if dec == nil {
		return data
	}
	decoded, _, err := transform.Bytes(dec, data)
	if err != nil {
		return data
	}
	return decoded
}

func decoderForCP(cp int) *encoding.Decoder {
	switch cp {
	case 437:
		return charmap.CodePage437.NewDecoder()
	case 850:
		return charmap.CodePage850.NewDecoder()
	case 852:
		return charmap.CodePage852.NewDecoder()
	case 855:
		return charmap.CodePage855.NewDecoder()
	case 858:
		return charmap.CodePage858.NewDecoder()
	case 860:
		return charmap.CodePage860.NewDecoder()
	case 862:
		return charmap.CodePage862.NewDecoder()
	case 863:
		return charmap.CodePage863.NewDecoder()
	case 865:
		return charmap.CodePage865.NewDecoder()
	case 866:
		return charmap.CodePage866.NewDecoder()
	case 932:
		return japanese.ShiftJIS.NewDecoder()
	case 936:
		return simplifiedchinese.GBK.NewDecoder()
	case 949:
		return korean.EUCKR.NewDecoder()
	case 950:
		return traditionalchinese.Big5.NewDecoder()
	case 1250:
		return charmap.Windows1250.NewDecoder()
	case 1251:
		return charmap.Windows1251.NewDecoder()
	case 1252:
		return charmap.Windows1252.NewDecoder()
	case 1253:
		return charmap.Windows1253.NewDecoder()
	case 1254:
		return charmap.Windows1254.NewDecoder()
	case 1255:
		return charmap.Windows1255.NewDecoder()
	case 1256:
		return charmap.Windows1256.NewDecoder()
	case 1257:
		return charmap.Windows1257.NewDecoder()
	case 1258:
		return charmap.Windows1258.NewDecoder()
	default:
		return nil
	}
}

func buildWindowsEnvStr() []uint16 {
	env := os.Environ()
	hasLang, hasTerm := false, false
	for _, e := range env {
		if strings.HasPrefix(e, "LANG=") {
			hasLang = true
		}
		if strings.HasPrefix(e, "TERM=") {
			hasTerm = true
		}
	}
	if !hasLang {
		env = append(env, "LANG=en_US.UTF-8")
	}
	if !hasTerm {
		env = append(env, "TERM=xterm-256color")
	}

	var buf []uint16
	for _, e := range env {
		buf = append(buf, windows.StringToUTF16(e)...)
		buf = append(buf, 0)
	}
	buf = append(buf, 0)
	return buf
}

// ---------- Session creation ----------

func NewSession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	session, err := sessionFromPipes(shellCommand, workDir)
	if err == nil {
		return session, nil
	}

	ptty, err := tryConPty(shellCommand, workDir, rows, cols)
	if err == nil {
		return sessionFromPty(ptty), nil
	}

	return nil, err
}

func tryConPty(shellCommand, workDir string, rows, cols int) (io.ReadWriteCloser, error) {
	if shellCommand == "" {
		shellCommand = DefaultShell()
	}
	cp, err := newConPty(int16(cols), int16(rows))
	if err != nil {
		return nil, err
	}
	if err := cp.startProcess(shellCommand, workDir); err != nil {
		cp.Close()
		return nil, err
	}
	return cp, nil
}

func sessionFromPty(ptty io.ReadWriteCloser) *Session {
	output := make(chan []byte, 256)
	done := make(chan struct{})
	session := &Session{ptty: ptty, output: output, done: done}

	go session.readLoop()

	if c, ok := ptty.(*conPty); ok {
		go func() {
			windows.WaitForSingleObject(c.procHandle, windows.INFINITE)
			close(done)
		}()
	}
	return session
}

func sessionFromPipes(shellCommand, workDir string) (*Session, error) {
	if shellCommand == "" {
		shellCommand = DefaultShell()
	}

	cmd := exec.Command(shellCommand)
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
	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8", "TERM=xterm-256color")

	if err := cmd.Start(); err != nil {
		stdinR.Close()
		stdinW.Close()
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	stdinR.Close()
	stdoutW.Close()

	// Decode OEM/ANSI codepage to UTF-8
	cp := getOEMCP()
	if cp == 65001 || cp == 0 {
		cp = getACP()
	}
	var stdout io.Reader = stdoutR
	if cp != 65001 && cp != 0 {
		dec := decoderForCP(cp)
		if dec != nil {
			stdout = transform.NewReader(stdoutR, dec)
		}
	}

	output := make(chan []byte, 256)
	done := make(chan struct{})

	ps := &pipeSession{stdin: stdinW, stdout: stdout}
	ptty := &lineWriter{ptty: ps, output: output}

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
	if c, ok := s.ptty.(*conPty); ok {
		return c.Resize(int16(cols), int16(rows))
	}
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
