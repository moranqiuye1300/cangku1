package events

const TopicVideoTranscode = "video.transcode"

type TranscodeTask struct {
	VideoID     string `json:"video_id"`
	UserID      string `json:"user_id"`
	SourcePath  string `json:"source_path"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
