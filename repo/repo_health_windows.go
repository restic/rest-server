package repo

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/itchio/ox/syscallex"
	"github.com/itchio/ox/winox"
	"golang.org/x/sys/windows"
)

var (
	advapi32DLL         = syscall.NewLazyDLL("advapi32.dll")
	impersonateSelfProc = advapi32DLL.NewProc("ImpersonateSelf")

	kernel32DLL             = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceExWProc = kernel32DLL.NewProc("GetDiskFreeSpaceExW")
)

func ImpersonateSelf(impersonationLevel uint64) (bool, error) {
	r1, _, err := impersonateSelfProc.Call(
		uintptr(impersonationLevel),
	)
	if err != syscall.Errno(0) {
		return false, err
	}
	return syscall.Handle(r1) == 1, nil
}

func GetDiskFreeSpaceExW(path string) (int64, error) {
	var freeBytes int64

	_, _, err := getDiskFreeSpaceExWProc.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&freeBytes)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		return 0, err
	}
	return freeBytes, nil
}

func isWritable(path string) (bool, error) {
	if _, err := ImpersonateSelf(syscallex.SecurityImpersonation); err != nil {
		return false, err
	}

	defer func() {
		_ = syscallex.RevertToSelf()
	}()

	var impersonationToken syscall.Token
	currentThread := syscallex.GetCurrentThread()
	err := syscallex.OpenThreadToken(
		currentThread,
		syscall.TOKEN_ALL_ACCESS,
		1,
		&impersonationToken,
	)
	if err != nil {
		return false, err
	}

	return winox.UserHasPermission(impersonationToken, winox.RightsRead|winox.RightsWrite, path)
}

func getFreeSpace(path string) (uint64, error) {
	freeBytes, err := GetDiskFreeSpaceExW(path)
	if err == nil && freeBytes < 0 {
		return 0, errors.New("free space can't be negative")
	}
	return uint64(freeBytes), err
}
