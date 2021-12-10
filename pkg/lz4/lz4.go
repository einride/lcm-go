package lz4

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/pierrec/lz4/v4"
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
	if err := comp.writer.Apply(lz4.BlockSizeOption(lz4.Block64Kb), lz4.DefaultConcurrency); err != nil {
		return nil
	}
	return comp
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	c.buffer.Reset()
	c.writer.Reset(c.buffer)
	if _, err := c.writer.Write(data); err != nil {
		return nil, fmt.Errorf("lz4 compress write: %w", err)
	}
	if err := c.writer.Close(); err != nil {
		return nil, fmt.Errorf("lz4 compress close: %w", err)
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
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("lz4 decompress read: %w", err)
	}
	return d.buf[:n], nil
}
