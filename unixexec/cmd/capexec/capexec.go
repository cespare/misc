package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// capabilities is the kernel struct described in 'man 2 capget'.
type capabilities struct {
	hdr struct {
		version uint32
		pid     int
	}
	data [2]struct {
		effective   uint32
		permitted   uint32
		inheritable uint32
	}
}

func main() {
	log.SetFlags(0)
	// FIXME(caleb): Implement the colon-based uid/gid syntax from chpst.
	username := flag.String("user", "", "User to run process as")
	capNames := flag.String("caps", "", "Add these capabilities to process (comma-separated list)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Usage: capexec [FLAGS] command args...")
	}

	var uid, gid uintptr
	if *username != "" {
		u, err := user.Lookup(*username)
		if err != nil {
			log.Fatalf("Unknown user %s", *username)
		}
		uid64, err := strconv.ParseInt(u.Uid, 0, 32)
		if err != nil {
			panic(err)
		}
		uid = uintptr(uid64)
		gid64, err := strconv.ParseInt(u.Gid, 0, 32)
		if err != nil {
			panic(err)
		}
		gid = uintptr(gid64)
	}

	caps := strings.Split(*capNames, ",")
	capIDs := make([]uintptr, len(caps))
	for i, cap := range caps {
		var id uintptr
		// Capabilities are defined in uapi/linux/capability.h.
		switch cap {
		case "net_bind_service":
			id = 10
		default:
			log.Fatalf("Unknown capability %q", cap)
		}
		capIDs[i] = id
	}

	argv0, err := exec.LookPath(args[0])
	if err != nil {
		log.Fatalf("Cannot find %s: %s", args[0], err)
	}

	envv := os.Environ()

	runtime.LockOSThread()

	if len(capIDs) > 0 {
		// Add capabilities to our permitted/inheritable sets or we
		// won't be able to add them to the ambient set later.
		var caps capabilities
		// Get the capability version (necessary for following syscall).
		_, _, errno := unix.RawSyscall(
			unix.SYS_CAPGET,
			uintptr(unsafe.Pointer(&caps.hdr)),
			0,
			0,
		)
		if errno != 0 {
			log.Fatalln("Error getting capability version:", errno)
		}
		// Get the current capabilities.
		_, _, errno = unix.RawSyscall(
			unix.SYS_CAPGET,
			uintptr(unsafe.Pointer(&caps.hdr)),
			uintptr(unsafe.Pointer(&caps.data[0])),
			0,
		)
		if errno != 0 {
			log.Fatalln("Error getting current capabilities:", errno)
		}
		for _, capID := range capIDs {
			caps.data[0].permitted |= 1 << uint(capID)
			caps.data[0].inheritable |= 1 << uint(capID)
		}
		_, _, errno = unix.RawSyscall(
			unix.SYS_CAPSET,
			uintptr(unsafe.Pointer(&caps.hdr)),
			uintptr(unsafe.Pointer(&caps.data[0])),
			0,
		)
		if errno != 0 {
			log.Fatalln("Error adding capabilities to effective/inheritable sets:", errno)
		}

		// We need to set the "keep capabilities" bit so we can set
		// ambient capabilities after a setuid.
		if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0); err != nil {
			log.Fatalln("Error setting KEEPCAPS:", err)
		}
	}

	if *username != "" {
		// {syscall,unix}.Setuid and {syscall,unix}.Setgid are disabled
		// because they're confusing (they only apply to the current thread).
		// We account for that here; do it ourselves.
		// See https://golang.org/issue/1435 for background info.
		if _, _, err := unix.RawSyscall(unix.SYS_SETGID, gid, 0, 0); err != 0 {
			log.Fatalln("Error setting GID:", err)
		}
		if _, _, err := unix.RawSyscall(unix.SYS_SETUID, uid, 0, 0); err != 0 {
			log.Fatalln("Error setting UID:", err)
		}
	}

	for _, capID := range capIDs {
		if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_RAISE, capID, 0, 0); err != nil {
			log.Fatalln("Error setting capability:", err)
		}
	}

	log.Fatal(unix.Exec(argv0, args, envv))
}
