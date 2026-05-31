package model

type AuditLog struct {
	ID             string
	Action         string
	ActorID        string
	ActorUsername  string
	TargetType     string
	TargetID       string
	IP             string
	UserAgent      string
	Detail         string
	CreatedAt      int64
}

type AuditLogDoc struct {
	LogID          string `bson:"log_id"`
	Action         string `bson:"action"`
	ActorID        string `bson:"actor_id"`
	ActorUsername  string `bson:"actor_username"`
	TargetType     string `bson:"target_type"`
	TargetID       string `bson:"target_id"`
	IP             string `bson:"ip"`
	UserAgent      string `bson:"user_agent"`
	Detail         string `bson:"detail"`
	CreatedAt      int64  `bson:"created_at"`
}

type VideoArchive struct {
	VideoID        string
	VideoSnapshot  Video
	LikeCount      int64
	CommentCount   int64
	FavoriteCount  int64
	DeletedAt      int64
	DeletedBy      string
	DeleteReason   string
	RetentionUntil int64
}

type VideoArchiveDoc struct {
	VideoID        string    `bson:"video_id"`
	VideoSnapshot  VideoDoc  `bson:"video_snapshot"`
	LikeCount      int64     `bson:"like_count"`
	CommentCount   int64     `bson:"comment_count"`
	FavoriteCount  int64     `bson:"favorite_count"`
	DeletedAt      int64     `bson:"deleted_at"`
	DeletedBy      string    `bson:"deleted_by"`
	DeleteReason   string    `bson:"delete_reason"`
	RetentionUntil int64     `bson:"retention_until"`
}
