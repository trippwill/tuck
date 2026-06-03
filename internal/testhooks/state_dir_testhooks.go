//go:build tuck_testhooks

package testhooks

import "os"

func testStateHomeOverride() string {
	return os.Getenv("TUCK_TEST_STATE_DIR")
}
