package app

type App struct {
	BorgmaticConfigs []string
	Port             string
}

type RepoInfo struct {
	Archives []struct {
		ChunkerParams []interface{} `json:"chunker_params"`
		CommandLine   []string      `json:"command_line"`
		Comment       string        `json:"comment"`
		Duration      float64       `json:"duration"`
		End           string        `json:"end"`
		Hostname      string        `json:"hostname"`
		Id            string        `json:"id"`
		Limits        struct {
			MaxArchiveSize float64 `json:"max_archive_size"`
		} `json:"limits"`
		Name  string `json:"name"`
		Start string `json:"start"`
		Stats struct {
			CompressedSize   int `json:"compressed_size"`
			DeduplicatedSize int `json:"deduplicated_size"`
			Nfiles           int `json:"nfiles"`
			OriginalSize     int `json:"original_size"`
		} `json:"stats"`
		Username string `json:"username"`
	} `json:"archives"`
	Cache struct {
		Path  string `json:"path"`
		Stats struct {
			TotalChunks       int   `json:"total_chunks"`
			TotalCsize        int64 `json:"total_csize"`
			TotalSize         int64 `json:"total_size"`
			TotalUniqueChunks int   `json:"total_unique_chunks"`
			UniqueCsize       int64 `json:"unique_csize"`
			UniqueSize        int64 `json:"unique_size"`
		} `json:"stats"`
	} `json:"cache"`
	Encryption struct {
		Mode string `json:"mode"`
	} `json:"encryption"`
	Repository struct {
		Id           string `json:"id"`
		LastModified string `json:"last_modified"`
		Location     string `json:"location"`
		Label        string `json:"label"`
	} `json:"repository"`
}
type RepoInfos []RepoInfo

// ListArchive - List archive
type ListArchive struct {
	Archives []struct {
		Archive  string `json:"archive"`
		Barchive string `json:"barchive"`
		Id       string `json:"id"`
		Name     string `json:"name"`
		Start    string `json:"start"`
		Time     string `json:"time"`
	} `json:"archives"`
	Encryption struct {
		Mode string `json:"mode"`
	} `json:"encryption"`
	Repository struct {
		Id           string `json:"id"`
		LastModified string `json:"last_modified"`
		Location     string `json:"location"`
		Label        string `json:"label"`
	} `json:"repository"`
}
type ListArchives []ListArchive
