package model

type Barrage struct {
	ID        string
	VideoID   string
	UserID    string
	Username  string
	Content   string
	TimeMs    int32
	CreatedAt int64
}

type BarrageDoc struct {
	BarrageID string `bson:"barrage_id"`
	VideoID   string `bson:"video_id"`
	UserID    string `bson:"user_id"`
	Username  string `bson:"username"`
	Content   string `bson:"content"`
	TimeMs    int32  `bson:"time_ms"`
	CreatedAt int64  `bson:"created_at"`
}

func DocToBarrage(d *BarrageDoc) *Barrage {
	return &Barrage{
		ID:        d.BarrageID,
		VideoID:   d.VideoID,
		UserID:    d.UserID,
		Username:  d.Username,
		Content:   d.Content,
		TimeMs:    d.TimeMs,
		CreatedAt: d.CreatedAt,
	}
}

func ToBarrageDoc(b Barrage) BarrageDoc {
	return BarrageDoc{
		BarrageID: b.ID,
		VideoID:   b.VideoID,
		UserID:    b.UserID,
		Username:  b.Username,
		Content:   b.Content,
		TimeMs:    b.TimeMs,
		CreatedAt: b.CreatedAt,
	}
}
