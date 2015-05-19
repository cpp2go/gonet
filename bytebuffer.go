package gonet

import (
	"encoding/binary"
)

const (
	presize  = 0
	initsize = 10
)

type ByteBuffer struct {
	_buffer      []byte
	_prependSize int
	_readerIndex int
	_writerIndex int
}

func NewByteBuffer() *ByteBuffer {
	return &ByteBuffer{
		_buffer:      make([]byte, presize+initsize),
		_prependSize: presize,
		_readerIndex: presize,
		_writerIndex: presize,
	}
}

func (this *ByteBuffer) Append(buff ...byte) {
	size := len(buff)
	if size > this.WrSize() {
		this.wrreserve(size)
	}
	copy(this._buffer[this._writerIndex:], buff)
	this.WrFlip(size)
}

func (this *ByteBuffer) WrBuf() []byte {
	if this._writerIndex >= len(this._buffer) {
		return nil
	}
	return this._buffer[this._writerIndex:]
}

func (this *ByteBuffer) WrSize() int {
	return len(this._buffer) - this._writerIndex
}

func (this *ByteBuffer) WrFlip(size int) {
	this._writerIndex += size
}

func (this *ByteBuffer) RdBuf() []byte {
	if this._readerIndex >= len(this._buffer) {
		return nil
	}
	return this._buffer[this._readerIndex:]
}

func (this *ByteBuffer) RdReady() bool {
	return this._writerIndex > this._readerIndex
}

func (this *ByteBuffer) RdSize() int {
	return this._writerIndex - this._readerIndex
}

func (this *ByteBuffer) RdFlip(size int) {
	if size < this.RdSize() {
		this._readerIndex += size
	} else {
		this.Reset()
	}
}

func (this *ByteBuffer) Reset() {
	this._readerIndex = this._prependSize
	this._writerIndex = this._prependSize
}

func (this *ByteBuffer) MaxSize() int {
	return len(this._buffer)
}

func (this *ByteBuffer) wrreserve(size int) {
	if this.WrSize()+this._readerIndex < size+this._prependSize {
		tmpbuff := make([]byte, this._writerIndex+size)
		copy(tmpbuff, this._buffer)
		this._buffer = tmpbuff
	} else {
		readable := this.RdSize()
		copy(this._buffer[this._prependSize:], this._buffer[this._readerIndex:this._writerIndex])
		this._readerIndex = this._prependSize
		this._writerIndex = this._readerIndex + readable
	}
}

func (this *ByteBuffer) Prepend(buff []byte) bool {
	size := len(buff)
	if this._readerIndex < size {
		return false
	}
	this._readerIndex -= size
	copy(this._buffer[this._readerIndex:], buff)
	return true
}

// Slice some bytes from buffer.
func (this *ByteBuffer) Slice(n int) []byte {
	if n > this.RdSize() {
		return make([]byte, 0, n)
	}
	r := this.RdBuf()[:n]
	this.RdFlip(n)
	return r
}

// Read a uint8 value from buffer.
func (this *ByteBuffer) ReadUint8() uint8 {
	return uint8(this.Slice(1)[0])
}

// Read a uint16 value from buffer using little endian byte order.
func (this *ByteBuffer) ReadUint16LE() uint16 {
	return binary.LittleEndian.Uint16(this.Slice(2))
}

// Read a uint16 value from buffer using big endian byte order.
func (this *ByteBuffer) ReadUint16BE() uint16 {
	return binary.BigEndian.Uint16(this.Slice(2))
}

// Read a uint32 value from buffer using little endian byte order.
func (this *ByteBuffer) ReadUint32LE() uint32 {
	return binary.LittleEndian.Uint32(this.Slice(4))
}

// Read a uint32 value from buffer using big endian byte order.
func (this *ByteBuffer) ReadUint32BE() uint32 {
	return binary.BigEndian.Uint32(this.Slice(4))
}

// Read a uint64 value from buffer using little endian byte order.
func (this *ByteBuffer) ReadUint64LE() uint64 {
	return binary.LittleEndian.Uint64(this.Slice(8))
}

// Read a uint64 value from buffer using big endian byte order.
func (this *ByteBuffer) ReadUint64BE() uint64 {
	return binary.BigEndian.Uint64(this.Slice(8))
}

// Write a uint8 value into buffer.
func (this *ByteBuffer) WriteUint8(v uint8) {
	this.Append(byte(v))
}

// Write a uint16 value into buffer using little endian byte order.
func (this *ByteBuffer) WriteUint16LE(v uint16) {
	this.Append(byte(v), byte(v>>8))
}

// Write a uint16 value into buffer using big endian byte order.
func (this *ByteBuffer) WriteUint16BE(v uint16) {
	this.Append(byte(v>>8), byte(v))
}

// Write a uint32 value into buffer using little endian byte order.
func (this *ByteBuffer) WriteUint32LE(v uint32) {
	this.Append(byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// Write a uint32 value into buffer using big endian byte order.
func (this *ByteBuffer) WriteUint32BE(v uint32) {
	this.Append(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// Write a uint64 value into buffer using little endian byte order.
func (this *ByteBuffer) WriteUint64LE(v uint64) {
	this.Append(byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

// Write a uint64 value into buffer using big endian byte order.
func (this *ByteBuffer) WriteUint64BE(v uint64) {
	this.Append(byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
