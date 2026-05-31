package model

type VideoStats struct {
	VideoID       string
	LikeCount     int64
	CommentCount  int64
	FavoriteCount int64
}

type VideoStatsDoc struct {
	VideoID       string `bson:"video_id"`
	LikeCount     int64  `bson:"like_count"`
	CommentCount  int64  `bson:"comment_count"`
	FavoriteCount int64  `bson:"favorite_count"`
}

type LikeDoc struct {
	VideoID   string `bson:"video_id"`
	UserID    string `bson:"user_id"`
	CreatedAt int64  `bson:"created_at"`
}

type FavoriteDoc struct {
	VideoID   string `bson:"video_id"`
	UserID    string `bson:"user_id"`
	CreatedAt int64  `bson:"created_at"`
}

type Comment struct {
	ID        string
	VideoID   string
	UserID    string
	Username  string
	Content   string
	CreatedAt int64
}

type CommentDoc struct {
	CommentID string `bson:"comment_id"`
	VideoID   string `bson:"video_id"`
	UserID    string `bson:"user_id"`
	Username  string `bson:"username"`
	Content   string `bson:"content"`
	CreatedAt int64  `bson:"created_at"`
}

func DocToComment(d *CommentDoc) *Comment {
	return &Comment{
		ID:        d.CommentID,
		VideoID:   d.VideoID,
		UserID:    d.UserID,
		Username:  d.Username,
		Content:   d.Content,
		CreatedAt: d.CreatedAt,
	}
}

func ToCommentDoc(c Comment) CommentDoc {
	return CommentDoc{
		CommentID: c.ID,
		VideoID:   c.VideoID,
		UserID:    c.UserID,
		Username:  c.Username,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}

type Engagement struct {
	VideoID       string
	LikeCount     int64
	CommentCount  int64
	FavoriteCount int64
	Liked         bool
	Favorited     bool
}
