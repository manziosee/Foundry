package types

type ImageInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	BlobKey  string `json:"blob_key"`
	Checksum string `json:"checksum"`
	State    string `json:"state"`
	Created  string `json:"created_at"`
	Updated  string `json:"updated_at"`
}