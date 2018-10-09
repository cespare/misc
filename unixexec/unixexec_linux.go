package unixexec

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// Defined in linux/prctl.h starting with Linux 4.3.
const (
	xPR_CAP_AMBIENT       = 0x2f
	xPR_CAP_AMBIENT_RAISE = 0x2
)

func execProcess(name string, argv []string, attr *os.ProcAttr) error {
	if attr == nil {
		attr = new(os.ProcAttr)
	}
	env := attr.Env
	if env == nil {
		env = os.Environ()
	}
	sys := attr.Sys
	if sys == nil {
		sys = new(syscall.SysProcAttr)
	}

	var chroot *byte
	var err error
	if sys.Chroot != "" {
		chroot, err = syscall.BytePtrFromString(sys.Chroot)
		if err != nil {
			return err
		}
	}
	var dir *byte
	if attr.Dir != "" {
		dir, err = syscall.BytePtrFromString(attr.Dir)
		if err != nil {
			return err
		}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	syscall.ForkLock.Lock()
	defer syscall.ForkLock.Unlock()

	if len(sys.AmbientCaps) > 0 {
		_, _, err = syscall.RawSyscall6(SYS_PRCTL, PR_SET_KEEPCAPS, 1, 0, 0, 0, 0)
		if err != 0 {
			return err
		}
	}

	// TODO: sys.UidMappings, sys.GidMappings?

	if sys.Setsid {
		_, _, err := syscall.RawSyscall(syscall.SYS_SETSID, 0, 0, 0)
		if err != 0 {
			return err
		}
	}

	if sys.Setpgid || sys.Foreground {
		_, _, err := syscall.RawSyscall(syscall.SYS_SETPGID, 0, uintptr(sys.Pgid), 0)
		if err != 0 {
			return err
		}
	}

	if sys.Foreground {
		pgrp := int32(sys.Pgid)
		if pgrp == 0 {
			r, _, err := syscall.RawSyscall(syscall.SYS_GETPID, 0, 0, 0)
			if err != 0 {
				return err
			}
			pgrp = int32(r)
		}
		_, _, err := syscall.RawSyscall(
			syscall.SYS_IOCTL,
			uintptr(sys.Ctty),
			uintptr(syscall.TIOCSPGRP),
			uintptr(unsafe.Pointer(&pgrp)),
		)
		if err != 0 {
			return err
		}
	}

	// TODO: Unshare?

	if chroot != nil {
		_, _, err := syscall.RawSyscall(syscall.SYS_CHROOT, uintptr(unsafe.Pointer(chroot)), 0, 0)
		if err != 0 {
			return err
		}
	}

	if cred := sys.Credential; cred != nil {
		// TODO: Setgroups?
		_, _, err := syscall.RawSyscall(syscall.SYS_SETGID, uintptr(cred.Gid), 0, 0)
		if err != 0 {
			return err
		}
		_, _, err = syscall.RawSyscall(syscall.SYS_SETUID, uintptr(cred.Uid), 0, 0)
		if err != 0 {
			return err
		}
	}

	for _, c := range sys.AmbientCaps {
		_, _, err := syscall.RawSyscall6(syscall.SYS_PRCTL, xPR_CAP_AMBIENT, xPR_CAP_AMBIENT_RAISE, c, 0, 0, 0)
		if err != 0 {
			return err
		}
	}

	if dir != nil {
		_, _, err := syscall.RawSyscall(syscall.SYS_CHDIR, uintptr(unsafe.Pointer(dir)), 0, 0)
		if err != 0 {
			return err
		}
	}

	// TODO: FD stuff?

	if sys.Noctty {
		_, _, err := syscall.RawSyscall(syscall.SYS_IOCTL, 0, syscall.TIOCNOTTY, 0)
		if err != 0 {
			return err
		}
	}

	if sys.Setctty {
		_, _, err := syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(sys.Ctty), syscall.TIOCSCTTY, 1)
		if err != 0 {
			return err
		}
	}

	if sys.Ptrace {
		_, _, err := syscall.RawSyscall(syscall.SYS_PTRACE, PTRACE_TRACEME, 0, 0)
		if err != 0 {
			return err
		}
	}

	return syscall.Exec(name, argv, env)
}
