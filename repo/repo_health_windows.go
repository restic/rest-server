package repo

import (
	"syscall"

	"github.com/itchio/ox/syscallex"
	"github.com/itchio/ox/winox"
)

var (
	advapi32DLL         = syscall.NewLazyDLL("advapi32.dll")
	impersonateSelfProc = advapi32DLL.NewProc("ImpersonateSelf")
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
