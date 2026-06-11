//go:build tuck_testhooks

package plan

import "os"

func hasRootPrivilege() bool {
	switch os.Getenv("TUCK_TEST_PRIVILEGE") {
	case "granted":
		return true
	case "denied":
		return false
	default:
		return os.Geteuid() == 0
	}
}
