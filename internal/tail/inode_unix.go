//go:build linux || darwin

package tail

import (
	"os"
	"syscall"
)

func inodeOf(path string) uint64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return stat.Ino
}
