package broadcast

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"context"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func nullableTime(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return s
}

func broadcastType(t string) string {
	switch t {
	case "popup", "banner":
		return t
	default:
		return "broadcast"
	}
}

func createBroadcast(c *gin.Context, user *auth.User, req createRequest) error {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	bType := broadcastType(req.Type)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if _, err := globals.ExecDb(db, `
		INSERT INTO broadcast (poster_id, content, type, start_at, end_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.GetID(db), req.Content, bType, nullableTime(req.StartAt), nullableTime(req.EndAt), isActive); err != nil {
		return err
	}

	cache.Del(context.Background(), ":broadcast")

	return nil
}

func updateBroadcast(c *gin.Context, req updateRequest) error {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	bType := broadcastType(req.Type)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if _, err := globals.ExecDb(db, `
		UPDATE broadcast SET content = ?, type = ?, start_at = ?, end_at = ?, is_active = ?
		WHERE id = ?
	`, req.Content, bType, nullableTime(req.StartAt), nullableTime(req.EndAt), isActive, req.Id); err != nil {
		return err
	}

	cache.Del(context.Background(), ":broadcast")

	return nil
}

func removeBroadcast(c *gin.Context, id int) error {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	if _, err := globals.ExecDb(db, `DELETE FROM broadcast WHERE id = ?`, id); err != nil {
		return err
	}

	cache.Del(context.Background(), ":broadcast")

	return nil
}

func getBroadcastList(c *gin.Context) ([]Info, error) {
	db := utils.GetDBFromContext(c)

	var broadcastList []Info
	rows, err := globals.QueryDb(db, `
		SELECT broadcast.id, broadcast.content, auth.username,
		       COALESCE(broadcast.type, 'broadcast'),
		       COALESCE(broadcast.start_at, ''), COALESCE(broadcast.end_at, ''),
		       COALESCE(broadcast.is_active, TRUE),
		       broadcast.created_at
		FROM broadcast
		INNER JOIN auth ON broadcast.poster_id = auth.id
		ORDER BY broadcast.id DESC
	`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var b Info
		var createdAt []uint8
		var startAt []uint8
		var endAt []uint8
		if err := rows.Scan(&b.Index, &b.Content, &b.Poster, &b.Type, &startAt, &endAt, &b.IsActive, &createdAt); err != nil {
			return nil, err
		}
		b.CreatedAt = utils.ConvertTime(createdAt).Format("2006-01-02 15:04:05")
		if t := utils.ConvertTime(startAt); t != nil && !t.IsZero() {
			b.StartAt = t.Format("2006-01-02 15:04:05")
		}
		if t := utils.ConvertTime(endAt); t != nil && !t.IsZero() {
			b.EndAt = t.Format("2006-01-02 15:04:05")
		}
		broadcastList = append(broadcastList, b)
	}

	return broadcastList, nil
}

func getLatestActiveBroadcast(c *gin.Context) *Broadcast {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	if data, err := cache.Get(context.Background(), ":broadcast").Result(); err == nil {
		if broadcast := utils.UnmarshalForm[Broadcast](data); broadcast != nil {
			return broadcast
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	var broadcast Broadcast
	if err := globals.QueryRowDb(db, `
		SELECT id, content FROM broadcast
		WHERE is_active = TRUE
		  AND (start_at IS NULL OR start_at <= ?)
		  AND (end_at IS NULL OR end_at >= ?)
		ORDER BY id DESC LIMIT 1
	`, now, now).Scan(&broadcast.Index, &broadcast.Content); err != nil {
		return nil
	}

	cache.Set(context.Background(), ":broadcast", utils.Marshal(broadcast), 10*time.Minute)
	return &broadcast
}
