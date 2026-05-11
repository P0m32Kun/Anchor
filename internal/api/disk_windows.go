//go:build windows

package api

import "golang.org/x/sys/windows"

type syscallStatfs struct {
	Bavail uint64
	Bsize  uint64
}

func statfs(path string, stat *syscallStatfs) error {
	var freeBytes, totalBytes, totalFreeBytes uint64
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	err = windows.GetDiskFreeSpaceEx(p, &freeBytes, &totalBytes, &totalFreeBytes)
	if err != nil {
		return err
	}
	stat.Bavail = freeBytes
	stat.Bsize = 1
	return nil
}
