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

type RouteManifestEntry interface {
	WeightedManifestEntry
	AddRoute(WeightedManifestEntry)
	GetRoutes() *[]WeightedManifestEntry
}

type MetadataFile struct {
	Title  string                  `json:"title"`
	Routes []WeightedManifestEntry `json:"routes"`
}

func NewMetadataFile() *MetadataFile {
	return &MetadataFile{
		Title:  "cert-manager",
		Routes: make([]WeightedManifestEntry, 0),
	}
}

func (m *MetadataFile) Weight() int64 {
	return 0
}

func (m *MetadataFile) AddRoute(w WeightedManifestEntry) {
	m.Routes = append(m.Routes, w)
}

func (m *MetadataFile) GetRoutes() *[]WeightedManifestEntry {
	return &m.Routes
}

func (m *MetadataFile) String() string {
	metadataFileInternal := struct {
		Routes []WeightedManifestEntry `json:"routes"`
	}{
		Routes: []WeightedManifestEntry{m},
	}

	out, _ := json.MarshalIndent(metadataFileInternal, "", "  ")
	return string(out)
}

type FileManifestEntry struct {
	Title     string `json:"title"`
	Path      string `json:"path"`
	RawWeight int64  `json:"-"`
}

func (f FileManifestEntry) Weight() int64 {
	return f.RawWeight
}

func (f FileManifestEntry) String() string {
	out, _ := json.MarshalIndent(f, "", "  ")
	return string(out)
}

type DirManifestEntry struct {
	Title  string                  `json:"title"`
	Routes []WeightedManifestEntry `json:"routes"`
}

func (d DirManifestEntry) Weight() int64 {
	return 0
}

func (d DirManifestEntry) String() string {
	sort.Slice(d.Routes, func(i, j int) bool {
		return d.Routes[i].Weight() < d.Routes[j].Weight()
	})

	out, _ := json.MarshalIndent(d, "", "  ")
	return string(out)
}

func (d *DirManifestEntry) AddRoute(w WeightedManifestEntry) {
	d.Routes = append(d.Routes, w)
}

func (m *DirManifestEntry) GetRoutes() *[]WeightedManifestEntry {
	return &m.Routes
}

type ManifestMetadata struct {
	Title  string `json:"title"`
	Weight int64  `json:"weight"`
}

func DeepestDir(m *MetadataFile, dirs []string) (*RouteManifestEntry, error) {
	return nil, fmt.Errorf("NYI")
}
