//go:build !windows

// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func terminalSize(file *os.File) (bool, int, int) {
	var size struct{ rows, columns, x, y uint16 }
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(),
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)))
	if err != 0 {
		return false, 80, 24
	}
	width, height := int(size.columns), int(size.rows)
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	return true, width, height
}
