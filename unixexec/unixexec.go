package unixexec

import (
	"os"
)

func ExecProcess(name string, argv []string, attr *os.ProcAttr) error {
	return execProcess(name, argv, attr)
}
