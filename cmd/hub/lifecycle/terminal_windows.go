// +build windows

package lifecycle

import (
	"io"
	"os"
)

func newTail(out *os.File) io.WriteCloser {
	return out
}
