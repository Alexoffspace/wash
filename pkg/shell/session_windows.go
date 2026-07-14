//go:build windows

package shell

import (
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
	procCreatePseudoConsole             = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole             = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole              = kernel32.NewProc("ClosePseudoConsole")
	procGetConsoleOutputCP              = kernel32.NewProc("GetConsoleOutputCP")
	procInitializeProcThreadAttributeList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute       = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList   = kernel32.NewProc("DeleteProcThreadAttributeList")
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
	hPC         windows.Handle
	hInputWrite windows.Handle
	hOutputRead windows.Handle
	attrListBuf []byte
	procHandle  windows.Handle
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
	normalized := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' && (i == 0 || b[i-1] != '\r') {
			normalized = append(normalized, '\r')
		} else {
			normalized = append(normalized, b[i])
		}
	}

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

func (c *conPty) sendEncodingInit(shellCommand string) {
	name := strings.ToLower(shellCommand)
	switch {
	case strings.Contains(name, "cmd"):
		c.Write([]byte("chcp 65001 > nul\r"))
	case strings.Contains(name, "powershell"):
		c.Write([]byte("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8\r"))
	}
}

func (c *conPty) Close() error {
	if c.hPC != 0 {
		procClosePseudoConsole.Call(uintptr(c.hPC))
		c.hPC = 0
	}
	if c.procHandle != 0 {
		windows.CloseHandle(c.procHandle)
		c.procHandle = 0
	}
	if c.hInputWrite != 0 {
		windows.CloseHandle(c.hInputWrite)
		c.hInputWrite = 0
	}
	if c.hOutputRead != 0 {
		windows.CloseHandle(c.hOutputRead)
		c.hOutputRead = 0
	}
	if len(c.attrListBuf) > 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&c.attrListBuf[0])))
		c.attrListBuf = nil
	}
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
		StartupInfo:   windows.StartupInfo{Flags: extendedStartupInfoPresent},
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

// ---------- Session creation ----------

func NewSession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	if shellCommand == "" {
		shellCommand = DefaultShell()
	}

	return newConPtySession(shellCommand, workDir, rows, cols)
}

func newConPtySession(shellCommand, workDir string, rows, cols int) (*Session, error) {
	cp, err := newConPty(int16(cols), int16(rows))
	if err != nil {
		return nil, err
	}

	if err := cp.startProcess(shellCommand, workDir); err != nil {
		cp.Close()
		return nil, err
	}

	cp.sendEncodingInit(shellCommand)

	output := make(chan []byte, 256)
	done := make(chan struct{})

	session := &Session{
		ptty:   cp,
		output: output,
		done:   done,
	}

	go session.readLoop()
	go waitProcessExit(cp, done)

	return session, nil
}

func waitProcessExit(cp *conPty, done chan struct{}) {
	defer close(done)
	windows.WaitForSingleObject(cp.procHandle, windows.INFINITE)
}

// ---------- Session methods ----------

func (s *Session) Resize(rows, cols int) error {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	if cp, ok := s.ptty.(*conPty); ok {
		return cp.Resize(int16(cols), int16(rows))
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
	}
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

// ---------- Resize helper ----------

func (c *conPty) Resize(cols, rows int16) error {
	procResizePseudoConsole.Call(uintptr(c.hPC), uintptr(uint32(uint16(rows))<<16|uint32(uint16(cols))))
	return nil
}

// ---------- Codepage helpers (for RunCommand) ----------

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
