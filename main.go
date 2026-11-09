package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spread/vaultplay/internal/index"
	"github.com/spread/vaultplay/internal/server"
)

//go:embed all:web/dist
var embeddedDist embed.FS

func main() {
	root := flag.String("root", "/videos", "media root directory")
	only := flag.String("only", "", "optional relative path of a single MP4 to index")
	listen := flag.String("listen", "unix:///run/vaultplay/http.sock", "listen address (unix:///path or tcp host:port)")
	flag.Parse()

	cat, err := index.Build(index.Options{Root: *root, Only: *only})
	if err != nil {
		log.Fatalf("index: %v", err)
	}
	log.Printf("indexed %d video(s) under %s", len(cat.List()), cat.Root())

	static, err := fs.Sub(embeddedDist, "web/dist")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}

	outbound := server.ProbeOutbound("1.1.1.1:443", 2*time.Second)
	if outbound.Reachable {
		log.Printf("warning: outbound TCP to %s succeeded (network not isolated)", outbound.Target)
	} else {
		log.Printf("outbound probe to %s failed as expected: %s", outbound.Target, outbound.Error)
	}

	srv := &server.Server{
		Catalog:  cat,
		Static:   static,
		Outbound: outbound,
	}

	ln, err := listenAddr(*listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("listening on %s", *listen)
	if err := http.Serve(ln, srv.Handler()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func listenAddr(addr string) (net.Listener, error) {
	switch {
	case strings.HasPrefix(addr, "unix://"):
		path := strings.TrimPrefix(addr, "unix://")
		if path == "" {
			return nil, fmt.Errorf("empty unix socket path")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		_ = os.Remove(path)
		ln, err := net.Listen("unix", path)
		if err != nil {
			return nil, err
		}
		if err := os.Chmod(path, 0o600); err != nil {
			_ = ln.Close()
			return nil, err
		}
		return ln, nil
	case strings.HasPrefix(addr, "tcp://"):
		return net.Listen("tcp", strings.TrimPrefix(addr, "tcp://"))
	default:
		// bare host:port
		return net.Listen("tcp", addr)
	}
}
