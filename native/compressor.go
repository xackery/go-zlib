package native

/*
#include "zlib.h"

// I have no idea why I have to wrap just this function but otherwise cgo won't compile
int defInit2(z_stream* s, int lvl, int method, int windowBits, int memLevel, int strategy) {
	return deflateInit2(s, lvl, method, windowBits, memLevel, strategy);
}
*/
import "C"
import (
	"fmt"
)

const defaultWindowBits = 15
const defaultMemLevel = 8

// Compressor using an underlying C zlib stream to compress (deflate) data
type Compressor struct {
	p     processor
	level int
}

// IsClosed returns whether the StreamCloser has closed the underlying stream
func (c *Compressor) IsClosed() bool {
	return c.p.isClosed
}

// NewCompressor returns and initializes a new Compressor with zlib compression stream initialized
func NewCompressor(lvl int) (*Compressor, error) {
	return NewCompressorStrategy(lvl, int(C.Z_DEFAULT_STRATEGY))
}

func NewCompressorRaw(lvl, strat, windowBits, memLevel int) (*Compressor, error) {
	p := newProcessor()

	if ok := C.defInit2(p.s, C.int(lvl), C.Z_DEFLATED, C.int(windowBits), C.int(memLevel), C.int(strat)); ok != C.Z_OK {
		return nil, determineError(fmt.Errorf("%s: %s", errInitialize.Error(), "compression level might be invalid"), ok)
	}

	return &Compressor{p, lvl}, nil
}

// NewCompressorStrategy returns and initializes a new Compressor with given level and strategy
// with zlib compression stream initialized
func NewCompressorStrategy(lvl, strat int) (*Compressor, error) {
	p := newProcessor()

	if ok := C.defInit2(p.s, C.int(lvl), C.Z_DEFLATED, C.int(defaultWindowBits), C.int(6), C.int(strat)); ok != C.Z_OK {
		return nil, determineError(fmt.Errorf("%s: %s", errInitialize.Error(), "compression level might be invalid"), ok)
	}

	return &Compressor{p, lvl}, nil
}

// Close closes the underlying zlib stream and frees the allocated memory
func (c *Compressor) Close() ([]byte, error) {
	condition := func() bool {
		return !c.p.hasCompleted
	}

	zlibProcess := func() C.int {
		return C.deflate(c.p.s, C.Z_FINISH)
	}

	_, b, err := c.p.process(
		[]byte{},
		[]byte{},
		condition,
		zlibProcess,
		func() C.int { return 0 },
	)

	ok := C.deflateEnd(c.p.s)

	c.p.close()

	if err != nil {
		return b, err
	}
	if ok != C.Z_OK {
		return b, determineError(errClose, ok)
	}

	return b, err
}

// Compress compresses the given data and returns it as byte slice
func (c *Compressor) Compress(in, out []byte) ([]byte, error) {
	zlibProcess := func() C.int {
		ok := C.deflate(c.p.s, C.Z_FINISH)
		if ok != C.Z_STREAM_END {
			return C.Z_BUF_ERROR
		}
		return ok
	}

	specificReset := func() C.int {
		return C.deflateReset(c.p.s)
	}

	_, b, err := c.p.process(
		in,
		out,
		nil,
		zlibProcess,
		specificReset,
	)
	return b, err
}

func (c *Compressor) CompressStream(in []byte) ([]byte, error) {
	zlibProcess := func() C.int {
		return C.deflate(c.p.s, C.Z_NO_FLUSH)
	}

	condition := func() bool {
		return c.p.getCompressed() == 0
	}

	_, b, err := c.p.process(
		in,
		make([]byte, 0, len(in)/assumedCompressionFactor),
		condition,
		zlibProcess,
		func() C.int { return 0 },
	)
	return b, err
}

// compress compresses the given data and returns it as byte slice
func (c *Compressor) compressFinish(in []byte) ([]byte, error) {
	condition := func() bool {
		return !c.p.hasCompleted
	}

	zlibProcess := func() C.int {
		return C.deflate(c.p.s, C.Z_FINISH)
	}

	specificReset := func() C.int {
		return C.deflateReset(c.p.s)
	}

	_, b, err := c.p.process(
		in,
		make([]byte, 0, len(in)/assumedCompressionFactor),
		condition,
		zlibProcess,
		specificReset,
	)
	return b, err
}

func (c *Compressor) Flush() ([]byte, error) {
	zlibProcess := func() C.int {
		return C.deflate(c.p.s, C.Z_SYNC_FLUSH)
	}

	condition := func() bool {
		return c.p.getCompressed() == 0
	}

	_, b, err := c.p.process(
		make([]byte, 0, 1),
		make([]byte, 0, 1),
		condition,
		zlibProcess,
		func() C.int { return 0 },
	)
	return b, err
}

func (c *Compressor) Reset() ([]byte, error) {
	b, err := c.compressFinish([]byte{})
	if err != nil {
		return b, err
	}

	return b, err
}
