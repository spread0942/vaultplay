// Package index recursively catalogs MP4 files under a media root.
package index

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spread/vaultplay/internal/mp4"
)

// Video is one indexed MP4 entry.
type Video struct {
	Name       string `json:"name"`
	RelPath    string `json:"relPath"`
	Dir        string `json:"dir"`
	Size       int64  `json:"size"`
	DurationMs *int64 `json:"durationMs"`
}

// Catalog is a path-safe in-memory index of MP4 files.
type Catalog struct {
	mu    sync.RWMutex
	root  string
	byRel map[string]Video
	list  []Video
}

// Options controls scanning.
type Options struct {
	Root string
	// Only, if non-empty, restricts the index to this relative path (slash-separated).
	Only string
}

// Build walks root and returns a catalog. only is a relative path filter (optional).
func Build(opts Options) (*Catalog, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	only := filepath.ToSlash(filepath.Clean(opts.Only))
	if only == "." {
		only = ""
	}

	c := &Catalog{
		root:  root,
		byRel: make(map[string]Video),
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.EqualFold(filepath.Ext(name), ".mp4") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if only != "" && rel != only {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		v := Video{
			Name:    name,
			RelPath: rel,
			Dir:     filepath.ToSlash(filepath.Dir(rel)),
			Size:    fi.Size(),
		}
		if v.Dir == "." {
			v.Dir = ""
		}
		if dur, err := mp4.Duration(path); err == nil && dur > 0 {
			ms := dur.Round(time.Millisecond).Milliseconds()
			v.DurationMs = &ms
		}
		c.byRel[rel] = v
		c.list = append(c.list, v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(c.list, func(i, j int) bool {
		if c.list[i].Dir != c.list[j].Dir {
			return c.list[i].Dir < c.list[j].Dir
		}
		return strings.ToLower(c.list[i].Name) < strings.ToLower(c.list[j].Name)
	})
	return c, nil
}

// Root returns the absolute media root.
func (c *Catalog) Root() string {
	return c.root
}

// List returns all videos in stable order.
func (c *Catalog) List() []Video {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Video, len(c.list))
	copy(out, c.list)
	return out
}

// Lookup returns a video by relative path if indexed.
func (c *Catalog) Lookup(rel string) (Video, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rel = filepath.ToSlash(filepath.Clean(rel))
	v, ok := c.byRel[rel]
	return v, ok
}

// ResolveAbs returns the absolute path for an indexed relative path, ensuring
// it stays inside the media root.
func (c *Catalog) ResolveAbs(rel string) (string, bool) {
	v, ok := c.Lookup(rel)
	if !ok {
		return "", false
	}
	abs := filepath.Join(c.root, filepath.FromSlash(v.RelPath))
	abs = filepath.Clean(abs)
	rootPrefix := c.root + string(os.PathSeparator)
	if abs != c.root && !strings.HasPrefix(abs, rootPrefix) {
		return "", false
	}
	return abs, true
}
