package output

import (
	"time"
)

type JSONFile struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"is_dir"`
	UpdateTime string `json:"update_time,omitempty"`
	FileID     string `json:"file_id"`
	PickCode   string `json:"pick_code,omitempty"`
	Sha1       string `json:"sha1,omitempty"`
}

func FileToJSON(f interface {
	GetName() string
	GetSize() int64
	IsDir() bool
	GetID() string
	ModTime() time.Time
}) JSONFile {
	return JSONFile{
		Name:       f.GetName(),
		Size:       f.GetSize(),
		IsDir:      f.IsDir(),
		UpdateTime: f.ModTime().Format(time.RFC3339),
		FileID:     f.GetID(),
	}
}

type JSONStat struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"is_dir"`
	FileID     string    `json:"file_id"`
	Sha1       string    `json:"sha1,omitempty"`
	PickCode   string    `json:"pick_code,omitempty"`
	CreateTime string    `json:"create_time"`
	UpdateTime string    `json:"update_time"`
	Parents    []JSONDir `json:"parents,omitempty"`
	FileCount  int       `json:"file_count,omitempty"`
	DirCount   int       `json:"dir_count,omitempty"`
}

type JSONDir struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
