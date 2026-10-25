package api

// BackupInput selects a new archive in the base's confined operational storage.
type BackupInput struct {
	Archive string `json:"archive"`
}

// RestoreInput never switches the running server's root. The destination must
// be a new directory under .kb/restores; existing directories are not reused.
type RestoreInput struct {
	Archive     string `json:"archive"`
	Destination string `json:"destination"`
}

// BackupResult describes an operational archive, not a knowledge export.
type BackupResult struct {
	Archive          string `json:"archive"`
	Destination      string `json:"destination,omitempty"`
	ManifestRevision string `json:"manifest_revision"`
	Files            int    `json:"files"`
	Bytes            int64  `json:"bytes"`
}
