// +build !linux

package unixexec

import (
	"errors"
	"os"
)

func execProcess(name string, argv []string, attr *os.ProcAttr) error {
	return errors.New("unixexec: ExecProcess not implemented for this OS")
}
