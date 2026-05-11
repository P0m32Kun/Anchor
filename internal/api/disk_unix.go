//go:build !windows

package api

import "golang.org/x/sys/unix"

type syscallStatfs unix.Statfs_t

func statfs(path string, stat *syscallStatfs) error {
	return unix.Statfs(path, (*unix.Statfs_t)(stat))
}
