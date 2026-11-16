package modbus

import (
	"encoding/binary"
	"errors"
	"io"
)

// ErrFrameLength is returned for an MBAP length field no frame can have (less
// than unit plus function code, or more than the largest PDU). The stream cannot
// be resynchronised after it, so the caller closes the connection.
var ErrFrameLength = errors.New("modbus: MBAP length out of range")

// Frame is one Modbus TCP application data unit.
type Frame struct {
	Transaction uint16
	Protocol    uint16 // 0 for Modbus; a frame of another protocol is discarded by the caller
	Unit        byte
	PDU         []byte // function code and data
}

// ReadFrame reads exactly one frame from r, however the bytes are split across
// reads. It returns io.EOF when r ends between frames and io.ErrUnexpectedEOF
// when it ends inside one.
func ReadFrame(r io.Reader) (Frame, error) {
	var head [headerSize]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return Frame{}, err
	}
	length := int(binary.BigEndian.Uint16(head[4:6]))
	if length < 2 || length > MaxPDU+1 {
		return Frame{}, ErrFrameLength
	}
	f := Frame{
		Transaction: binary.BigEndian.Uint16(head[0:2]),
		Protocol:    binary.BigEndian.Uint16(head[2:4]),
		Unit:        head[6],
		PDU:         make([]byte, length-1),
	}
	if _, err := io.ReadFull(r, f.PDU); err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return Frame{}, err
	}
	return f, nil
}

// Bytes is the frame as it goes on the wire.
func (f Frame) Bytes() []byte {
	out := make([]byte, headerSize, headerSize+len(f.PDU))
	binary.BigEndian.PutUint16(out[0:2], f.Transaction)
	binary.BigEndian.PutUint16(out[2:4], f.Protocol)
	binary.BigEndian.PutUint16(out[4:6], uint16(len(f.PDU)+1))
	out[6] = f.Unit
	return append(out, f.PDU...)
}
