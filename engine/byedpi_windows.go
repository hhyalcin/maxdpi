//go:build windows

package engine

import (
	_ "embed"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ciadpi.exe: scripts/build-engines.sh ile sabit commit'ten mingw-w64 ile kaynaktan derlenir.
//
//go:embed bin_windows/ciadpi.exe
var ciadpiBin []byte

const ciadpiName = "ciadpi.exe"

// DefaultProfiles: ByeDPI README'sine gore Windows'ta saf kullanici-alani yontemleri (tlsrec/split/oob) once,
// disorder/fake ("Windows'ta kararsiz olabilir") sonra denenir. Gercek Windows testi gerekir.
var DefaultProfiles = []Profile{
	{Name: "split+tlsrec", Args: []string{"--split", "2", "--tlsrec", "3+s"}},
	{Name: "oob", Args: []string{"--oob", "1"}},
	{Name: "disorder+tlsrec", Args: []string{"--disorder", "1", "--tlsrec", "1+s"}},
	{Name: "split+disorder", Args: []string{"--split", "1+s", "--disorder", "3+s"}},
	{Name: "disorder+fake", Args: []string{"--disorder", "1", "--fake", "-1"}},
}

func prepareCmd(cmd *exec.Cmd) {
	// Konsol penceresi acilmasin; ortam minimum (Winsock icin SystemRoot gerekir).
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot")}
}

var (
	jobOnce sync.Once
	job     windows.Handle
	jobErr  error
)

// attach: motoru "kapaninca oldur" job nesnesine baglar. Uygulama cokse/olse bile isletim sistemi motoru sonlandirir.
func attach(cmd *exec.Cmd) error {
	jobOnce.Do(func() {
		job, jobErr = windows.CreateJobObject(nil, nil)
		if jobErr != nil {
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		_, jobErr = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	})
	if jobErr != nil {
		return jobErr
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return windows.AssignProcessToJobObject(job, h)
}

// killStale: job nesnesi yetim motor birakmaz; ek is yok.
func killStale(string) {}
