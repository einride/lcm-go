package lz4

import (
	"bytes"

	"github.com/pierrec/lz4/v3"
	"golang.org/x/xerrors"
)

const maxMessageSize = 65565 * 2 // on the safe side

type Compressor struct {
	buffer *bytes.Buffer
	writer *lz4.Writer
}

type Decompressor struct {
	buf    []byte
	reader *lz4.Reader
}

func NewCompressor() *Compressor {
	comp := &Compressor{
		buffer: bytes.NewBuffer(nil),
		writer: lz4.NewWriter(nil),
	}
	comp.writer.Header.BlockMaxSize = 64 << 10
	comp.writer.Header.BlockChecksum = true
	return comp
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	c.buffer.Reset()
	c.writer.Reset(c.buffer)
	_, err := c.writer.Write(data)
	if err != nil {
		return nil, xerrors.Errorf("lz4 compress write: %w", err)
	}
	if err = c.writer.Flush(); err != nil {
		return nil, xerrors.Errorf("lz4 compress flush: %w", err)
	}
	return c.buffer.Bytes(), nil
}

func (c *Compressor) Name() string {
	return "lz4"
}

func NewDecompressor() *Decompressor {
	return &Decompressor{
		buf:    make([]byte, maxMessageSize),
		reader: lz4.NewReader(nil),
	}
}

func (d *Decompressor) Decompress(data []byte) ([]byte, error) {
	d.reader.Reset(bytes.NewBuffer(data))
	n, err := d.reader.Read(d.buf)
	if err != nil {
		return nil, xerrors.Errorf("lz4 decompress read: %w", err)
	}
	return d.buf[:n], nil
}
