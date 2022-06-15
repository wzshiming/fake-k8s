//go:build windows

package utils

import (
	"path/filepath"
	"strings"
)

func PathJoin(elem ...string) string {
	return strings.Replace(filepath.Join(elem...), `\`, `/`, -1)
}
