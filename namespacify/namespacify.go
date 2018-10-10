package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cespare/cp"
)

const reexecEnv = "NAMESPACIFY_REEXEC"

func main() {
	log.SetFlags(0)

	if chroot := os.Getenv(reexecEnv); chroot != "" {
		configureNamespace(chroot)
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		setStdIO(cmd)
		exitWithStatus(cmd.Run())
	}

	chrootDir := flag.String("dir", "chroot", "Directory for chroot")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatalf("usage: %s command [args...]", os.Args[0])
	}
	if *chrootDir == "" {
		log.Fatalln("-chroot cannot be empty")
	}
	if err := os.RemoveAll(*chrootDir); err != nil {
		if !os.IsExist(err) {
			log.Fatalln("Cannot clear chroot dir:", err)
		}
	}
	if err := os.MkdirAll(*chrootDir, 0755); err != nil {
		log.Fatalln("Cannot create chroot dir:", err)
	}
	cmd := &exec.Cmd{
		Path:        "/proc/self/exe",
		Args:        flag.Args(),
		Env:         append(os.Environ(), reexecEnv+"="+*chrootDir),
		SysProcAttr: &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM},
	}
	setStdIO(cmd)

	cloneFlags := syscall.CLONE_NEWIPC |
		syscall.CLONE_NEWNS |
		syscall.CLONE_NEWPID |
		syscall.CLONE_NEWUSER |
		syscall.CLONE_NEWUTS
	uidMappings := []syscall.SysProcIDMap{
		{
			ContainerID: 0,
			HostID:      os.Getuid(),
			Size:        1,
		},
	}
	gidMappings := []syscall.SysProcIDMap{
		{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		},
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  uintptr(cloneFlags),
		UidMappings: uidMappings,
		GidMappings: gidMappings,
	}

	exitWithStatus(cmd.Run())
}

func configureNamespace(chroot string) {
	for _, dir := range []string{
		"/bin",
		"/dev",
		"/lib",
		"/lib64",
		"/proc",
		"/sbin",
		"/sys",
		"/usr",
	} {
		target := filepath.Join(chroot, dir)
		if err := mkdir(target); err != nil {
			log.Fatal(err)
		}
		if err := mountBind(dir, target); err != nil {
			log.Fatal(err)
		}
	}
	for _, p := range []string{
		"/etc/resolv.conf",
		"/etc/ssl/certs",
		"/etc/passwd",
	} {
		dst := filepath.Join(chroot, p)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			log.Fatal(err)
		}
		if err := cp.CopyAll(dst, p); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.Chdir(chroot); err != nil {
		log.Fatal(err)
	}
	if err := syscall.Chroot("."); err != nil {
		log.Fatal(err)
	}
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		log.Fatal(err)
	}
	name := "ns-" + hex.EncodeToString(id)
	os.Setenv("PS1", name+"$ ")
	if err := syscall.Sethostname([]byte(name)); err != nil {
		log.Fatal(err)
	}
}

func mountBind(source, target string) error {
	return syscall.Mount(
		source,
		target,
		"", // ignored
		syscall.MS_BIND|syscall.MS_REC|syscall.MS_RDONLY,
		"", // ignored
	)
}

func mkdir(dir string) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func setStdIO(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func exitWithStatus(err error) {
	if err == nil {
		os.Exit(0)
	}
	e, ok := err.(*exec.ExitError)
	if !ok {
		log.Fatal(err)
	}
	ws, ok := e.Sys().(syscall.WaitStatus)
	if !ok {
		log.Fatal(err)
	}
	os.Exit(ws.ExitStatus())
}
