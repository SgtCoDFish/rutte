package rutte

type ManifestEntry struct {
	Title  string          `json:"title"`
	Path   string          `json:"path,omitempty"`
	Routes []ManifestEntry `json:"routes,omitempty"`
	Weight int64           `json:"-"`
}

type ManifestMetadata struct {
	Title  string `json:"title"`
	Weight int64  `json:"weight"`
}
