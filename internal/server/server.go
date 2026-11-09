// Package server serves the Vaultplay API, media streams, and embedded SPA.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spread/vaultplay/internal/index"
)

// Server is the HTTP front-end for a media catalog.
type Server struct {
	Catalog  *index.Catalog
	Static   fs.FS
	Outbound OutboundStatus
}

// OutboundStatus records whether the process can open outbound TCP.
type OutboundStatus struct {
	Attempted bool   `json:"attempted"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
	Target    string `json:"target"`
}

// ProbeOutbound tries a short TCP dial to demonstrate network isolation.
func ProbeOutbound(target string, timeout time.Duration) OutboundStatus {
	st := OutboundStatus{Attempted: true, Target: target}
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		st.Reachable = false
		st.Error = err.Error()
		return st
	}
	_ = conn.Close()
	st.Reachable = true
	return st
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/videos", s.handleVideos)
	mux.HandleFunc("/media/", s.handleMedia)
	mux.Handle("/", s.spaHandler())
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"videos":   len(s.Catalog.List()),
		"outbound": s.Outbound,
	})
}

func (s *Server) handleVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"videos": s.Catalog.List(),
	})
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/media/")
	rel = path.Clean("/" + rel)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		http.NotFound(w, r)
		return
	}
	abs, ok := s.Catalog.ResolveAbs(rel)
	if !ok {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
}

func (s *Server) spaHandler() http.Handler {
	if s.Static == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not embedded", http.StatusServiceUnavailable)
		})
	}
	fileServer := http.FileServer(http.FS(s.Static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" || p == "." {
			// Serve the app shell only at "/".
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		if _, err := fs.Stat(s.Static, p); err != nil {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		log.Printf("json encode: %v", err)
	}
}
