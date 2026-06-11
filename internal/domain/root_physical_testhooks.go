//go:build tuck_testhooks

package domain

import "os"

func rootPhysicalRoot() string {
	if root := os.Getenv("TUCK_TEST_ROOT_DIR"); root != "" {
		return root
	}
	return "/"
}
