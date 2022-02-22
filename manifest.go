package rutte

import (
	"encoding/json"
	"fmt"
	"sort"
)

type WeightedManifestEntry interface {
	fmt.Stringer
	Weight() int64
}

type FileManifestEntry struct {
	Title  string `json:"title"`
	Path   string `json:"path"`
	Weight int64  `json:"-"`
}

var _ WeightedManifestEntry = (*FileManifestEntry)(nil)

func (f FileManifestEntry) Weight() int64 {
	return f.Weight
}

func (f FileManifestEntry) String() string {
	out, _ := json.MarshalIndent(f, "", "  ")
	return string(out)
}

type DirManifestEntry struct {
	Title  string              `json:"title"`
	Routes []FileManifestEntry `json:"routes"`
}

var _ WeightedManifestEntry = (*DirManifestEntry)(nil)

func (d DirManifestEntry) Weight() int64 {
	return 0
}

func (d DirManifestEntry) String() string {
	sort.Slice(d.Routes, func(i, j int) bool {
		return d.Routes[i].Weight() < d.Routes[j].Weight()
	})
}
