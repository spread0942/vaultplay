package index

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func writeMinimalMP4(t *testing.T, path string, timescale, duration uint32) {
	t.Helper()
	buf := make([]byte, 0, 256)
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
	binary.BigEndian.PutUint32(mvhd[12:16], timescale)
	binary.BigEndian.PutUint32(mvhd[16:20], duration)
	var moov []byte
	{
		size := uint32(8 + len(mvhd))
		var hdr [8]byte
		binary.BigEndian.PutUint32(hdr[0:4], size)
		copy(hdr[4:8], "mvhd")
		moov = append(moov, hdr[:]...)
		moov = append(moov, mvhd...)
	}
	writeBox("moov", moov)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAndResolve(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "Clips")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMinimalMP4(t, filepath.Join(dir, "Root.MP4"), 1000, 2000)
	writeMinimalMP4(t, filepath.Join(sub, "nested.mp4"), 1000, 3500)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("nope"), 0o644)

	cat, err := Build(Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.List()) != 2 {
		t.Fatalf("want 2 videos, got %d", len(cat.List()))
	}

	abs, ok := cat.ResolveAbs("Clips/nested.mp4")
	if !ok {
		t.Fatal("expected nested lookup")
	}
	if abs != filepath.Join(dir, "Clips", "nested.mp4") {
		t.Fatalf("unexpected abs %s", abs)
	}
	if _, ok := cat.ResolveAbs("../Root.MP4"); ok {
		t.Fatal("path traversal must fail")
	}
	if _, ok := cat.ResolveAbs("missing.mp4"); ok {
		t.Fatal("missing must fail")
	}

	only, err := Build(Options{Root: dir, Only: "Root.MP4"})
	if err != nil {
		t.Fatal(err)
	}
	if len(only.List()) != 1 || only.List()[0].RelPath != "Root.MP4" {
		t.Fatalf("only filter failed: %+v", only.List())
	}
}
