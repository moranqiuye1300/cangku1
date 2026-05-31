package model

const (
	StatusPending     = "pending"
	StatusTranscoding = "transcoding"
	StatusReady       = "ready"
	StatusFailed      = "failed"

	RecycleBinDays = 30
)

type Video struct {
	ID           string
	UserID       string
	Title        string
	Description  string
	CoverURL     string
	Status       string
	Duration     int32
	CreatedAt    int64
	PlayURLs     map[string]string
	Tags         []string
	SourcePath   string
	DeletedAt    int64
	DeletedBy    string
	DeleteReason string
	PurgeAt      int64
}

type VideoDoc struct {
	VideoID      string            `bson:"video_id"`
	UserID       string            `bson:"user_id"`
	Title        string            `bson:"title"`
	Description  string            `bson:"description"`
	CoverURL     string            `bson:"cover_url"`
	Status       string            `bson:"status"`
	Duration     int32             `bson:"duration"`
	CreatedAt    int64             `bson:"created_at"`
	PlayURLs     map[string]string `bson:"play_urls,omitempty"`
	Tags         []string          `bson:"tags,omitempty"`
	SourcePath   string            `bson:"source_path,omitempty"`
	DeletedAt    int64             `bson:"deleted_at,omitempty"`
	DeletedBy    string            `bson:"deleted_by,omitempty"`
	DeleteReason string            `bson:"delete_reason,omitempty"`
	PurgeAt      int64             `bson:"purge_at,omitempty"`
}

func DocToVideo(d *VideoDoc) *Video {
	urls := d.PlayURLs
	if urls == nil {
		urls = map[string]string{}
	}
	return &Video{
		ID:           d.VideoID,
		UserID:       d.UserID,
		Title:        d.Title,
		Description:  d.Description,
		CoverURL:     d.CoverURL,
		Status:       d.Status,
		Duration:     d.Duration,
		CreatedAt:    d.CreatedAt,
		PlayURLs:     urls,
		Tags:         d.Tags,
		SourcePath:   d.SourcePath,
		DeletedAt:    d.DeletedAt,
		DeletedBy:    d.DeletedBy,
		DeleteReason: d.DeleteReason,
		PurgeAt:      d.PurgeAt,
	}
}

func ToDoc(v Video) VideoDoc {
	urls := v.PlayURLs
	if urls == nil {
		urls = map[string]string{}
	}
	return VideoDoc{
		VideoID:      v.ID,
		UserID:       v.UserID,
		Title:        v.Title,
		Description:  v.Description,
		CoverURL:     v.CoverURL,
		Status:       v.Status,
		Duration:     v.Duration,
		CreatedAt:    v.CreatedAt,
		PlayURLs:     urls,
		Tags:         v.Tags,
		SourcePath:   v.SourcePath,
		DeletedAt:    v.DeletedAt,
		DeletedBy:    v.DeletedBy,
		DeleteReason: v.DeleteReason,
		PurgeAt:      v.PurgeAt,
	}
}
