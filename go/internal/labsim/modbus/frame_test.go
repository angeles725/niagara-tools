package modbus

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
	"testing/iotest"
)

// A read of one holding register, unit 0x11, transaction 0x1501: the MBAP
// header layout of the Modbus Messaging on TCP/IP Implementation Guide V1.0b,
// section 3.1.3 (transaction, protocol 0, length = unit + PDU, unit).
var exampleADU = []byte{0x15, 0x01, 0x00, 0x00, 0x00, 0x06, 0x11, 0x03, 0x00, 0x6B, 0x00, 0x03}

func TestReadFrameSplitsTheHeaderFromThePDU(t *testing.T) {
	t.Parallel()
	got, err := ReadFrame(bytes.NewReader(exampleADU))
	want := Frame{Transaction: 0x1501, Unit: 0x11, PDU: []byte{0x03, 0x00, 0x6B, 0x00, 0x03}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadFrame = %+v, %v; want %+v", got, err, want)
	}
	if back := got.Bytes(); !bytes.Equal(back, exampleADU) {
		t.Errorf("Bytes() = % X, want % X", back, exampleADU)
	}
}

func TestReadFrameJoinsAFrameThatArrivesByteByByte(t *testing.T) {
	t.Parallel()
	two := append(append([]byte{}, exampleADU...), exampleADU...)
	r := iotest.OneByteReader(bytes.NewReader(two))
	for i := range 2 {
		if f, err := ReadFrame(r); err != nil || f.Transaction != 0x1501 || len(f.PDU) != 5 {
			t.Fatalf("frame %d = %+v, %v", i, f, err)
		}
	}
	if _, err := ReadFrame(r); !errors.Is(err, io.EOF) {
		t.Errorf("after the last frame: %v, want io.EOF", err)
	}
}

func TestReadFrameRefusesALengthNoFrameCanHave(t *testing.T) {
	t.Parallel()
	for _, length := range []uint16{0, 1, 255, 0xFFFF} {
		head := []byte{0, 1, 0, 0, byte(length >> 8), byte(length), 1}
		if _, err := ReadFrame(bytes.NewReader(append(head, make([]byte, 300)...))); !errors.Is(err, ErrFrameLength) {
			t.Errorf("length %d: %v, want ErrFrameLength", length, err)
		}
	}
	largest := append([]byte{0, 1, 0, 0, 0, 254, 1}, make([]byte, 253)...)
	if f, err := ReadFrame(bytes.NewReader(largest)); err != nil || len(f.PDU) != MaxPDU {
		t.Errorf("length 254: %d PDU bytes, %v; want the largest PDU", len(f.PDU), err)
	}
}

func TestReadFrameReportsAFrameCutShort(t *testing.T) {
	t.Parallel()
	if _, err := ReadFrame(bytes.NewReader(exampleADU[:9])); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("cut PDU: %v, want io.ErrUnexpectedEOF", err)
	}
	if _, err := ReadFrame(bytes.NewReader(exampleADU[:3])); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("cut header: %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadFrameKeepsAForeignProtocolIdentifier(t *testing.T) {
	t.Parallel()
	adu := append([]byte{}, exampleADU...)
	adu[3] = 7
	if f, err := ReadFrame(bytes.NewReader(adu)); err != nil || f.Protocol != 7 {
		t.Errorf("ReadFrame = %+v, %v; want protocol 7 for the caller to discard", f, err)
	}
}
