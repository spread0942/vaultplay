// Package mp4 provides a minimal MP4 moov/mvhd duration reader (stdlib only).
package mp4

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"time"
)

var (
	errTruncated = errors.New("mp4: truncated box")
	errNoMVHD    = errors.New("mp4: mvhd not found")
)

// Duration opens path and returns the movie duration from the mvhd box.
func Duration(path string) (time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return durationFrom(f, info.Size())
}

func durationFrom(r io.ReaderAt, size int64) (time.Duration, error) {
	timescale, duration, err := findMVHD(r, 0, size)
	if err != nil {
		return 0, err
	}
	if timescale == 0 {
		return 0, errors.New("mp4: zero timescale")
	}
	return time.Duration(duration) * time.Second / time.Duration(timescale), nil
}

func findMVHD(r io.ReaderAt, start, end int64) (timescale uint32, duration uint64, err error) {
	off := start
	for off+8 <= end {
		var hdr [16]byte
		n := 8
		if _, err := r.ReadAt(hdr[:8], off); err != nil {
			return 0, 0, err
		}
		size := uint64(binary.BigEndian.Uint32(hdr[0:4]))
		typ := string(hdr[4:8])
		headerLen := int64(8)
		if size == 1 {
			if _, err := r.ReadAt(hdr[8:16], off+8); err != nil {
				return 0, 0, err
			}
			size = binary.BigEndian.Uint64(hdr[8:16])
			headerLen = 16
			n = 16
		} else if size == 0 {
			size = uint64(end - off)
		}
		_ = n
		if size < uint64(headerLen) {
			return 0, 0, errTruncated
		}
		boxEnd := off + int64(size)
		if boxEnd > end || boxEnd < off {
			return 0, 0, errTruncated
		}
		payloadStart := off + headerLen

		switch typ {
		case "moov", "trak", "mdia", "minf", "stbl", "edts":
			ts, dur, err := findMVHD(r, payloadStart, boxEnd)
			if err == nil {
				return ts, dur, nil
			}
			if !errors.Is(err, errNoMVHD) {
				return 0, 0, err
			}
		case "mvhd":
			return parseMVHD(r, payloadStart, boxEnd)
		}
		off = boxEnd
	}
	return 0, 0, errNoMVHD
}

func parseMVHD(r io.ReaderAt, start, end int64) (uint32, uint64, error) {
	// version(1) + flags(3) + creation/modification + timescale + duration
	need := int64(4 + 8 + 4 + 4) // v0 minimum after version/flags uses 4-byte times
	if end-start < need {
		return 0, 0, errTruncated
	}
	var buf [32]byte
	if _, err := r.ReadAt(buf[:1], start); err != nil {
		return 0, 0, err
	}
	version := buf[0]
	if version == 1 {
		if end-start < 4+16+4+8 {
			return 0, 0, errTruncated
		}
		if _, err := r.ReadAt(buf[:28], start+4); err != nil {
			return 0, 0, err
		}
		ts := binary.BigEndian.Uint32(buf[16:20])
		dur := binary.BigEndian.Uint64(buf[20:28])
		return ts, dur, nil
	}
	// version 0
	if _, err := r.ReadAt(buf[:20], start+4); err != nil {
		return 0, 0, err
	}
	ts := binary.BigEndian.Uint32(buf[8:12])
	dur := uint64(binary.BigEndian.Uint32(buf[12:16]))
	return ts, dur, nil
}
