package zlib

import "github.com/xackery/go-zlib/native"

func checkClosed(c native.StreamCloser) error {
	if c.IsClosed() {
		return errIsClosed
	}
	return nil
}
