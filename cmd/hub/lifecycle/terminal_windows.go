// +build windows

package lifecycle

import (
	"io"
	"os"
)

func newTail(out *os.File) io.Writer {
	return out
}
