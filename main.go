package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("help")
	}
}

func run() {
	fmt.Printf("Running %v as user %d in process %d\n",
		os.Args[2:],
		os.Geteuid(),
		os.Getpid())

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Clearenv()
	cmd.Env = []string{
		"PATH=/bin:/usr/bin",
		"PS1=[\\u@\\h \\w]\\$ ",
	}

	// set NameSpaces isolation
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // Unix Timesharing System (hostname)
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNS | // mount points
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER,
		// map the files to user root and allow running a rootless
		// container (with no mapping we'll end with the error message:
		// panic: fork/exec /proc/self/exe: operation not permitted)
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
		// do not share the mount points to the host
		// (will not be shown in the mount command output but
		// you can still see it in the /proc filesystem)
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("Running %v as user %d in process %d\n",
		os.Args[2:],
		os.Geteuid(),
		os.Getpid())

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	must(syscall.Chroot("./alpinefs"))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
