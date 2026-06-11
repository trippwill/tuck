//go:build !tuck_testhooks

package plan

import "os"

func hasRootPrivilege() bool {
	return os.Geteuid() == 0
}
