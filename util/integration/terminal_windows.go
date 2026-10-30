// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import "os"

func terminalSize(_ *os.File) (bool, int, int) {
	return false, 80, 24
}
