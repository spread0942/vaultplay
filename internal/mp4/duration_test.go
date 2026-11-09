package mp4

import (
	"encoding/binary"
	"io"
	"testing"
	"time"
)

func TestDurationFromSyntheticMVHD(t *testing.T) {
	// ftyp (minimal) + moov/mvhd v0: timescale=1000, duration=5500 => 5.5s
	buf := make([]byte, 0, 128)

	writeBox := func(typ string, payload []byte) {
		size := uint32(8 + len(payload))
		var hdr [8]byte
		binary.BigEndian.PutUint32(hdr[0:4], size)
		copy(hdr[4:8], typ)
		buf = append(buf, hdr[:]...)
		buf = append(buf, payload...)
	}

	writeBox("ftyp", []byte("isomisom"))

	mvhd := make([]byte, 100)
	mvhd[0] = 0 // version
	binary.BigEndian.PutUint32(mvhd[12:16], 1000)
	binary.BigEndian.PutUint32(mvhd[16:20], 5500)

	moovPayload := make([]byte, 0, 128)
	{
		size := uint32(8 + len(mvhd))
		var hdr [8]byte
		binary.BigEndian.PutUint32(hdr[0:4], size)
		copy(hdr[4:8], "mvhd")
		moovPayload = append(moovPayload, hdr[:]...)
		moovPayload = append(moovPayload, mvhd...)
	}
	writeBox("moov", moovPayload)

	d, err := durationFrom(bytesReaderAt(buf), int64(len(buf)))
	if err != nil {
		t.Fatal(err)
	}
	if d != 5500*time.Millisecond {
		t.Fatalf("got %v want 5.5s", d)
	}
}

type bytesReaderAt []byte

func (b bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errTruncated
	}
	if off >= int64(len(b)) {
		return 0, io.EOF
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
