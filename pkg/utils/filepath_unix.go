//go:build !windows

package utils

import (
	"path/filepath"
)

func PathJoin(elem ...string) string {
	return filepath.Join(elem...)
}
