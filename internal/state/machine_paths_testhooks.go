//go:build tuck_testhooks

package state

import "os"

func testStateHomeOverride() string {
	return os.Getenv("TUCK_TEST_STATE_DIR")
}
