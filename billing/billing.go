package billing

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const pageSize int64 = 20

const recordFilterWhereSQL = `
		WHERE (? = 0 OR b.user_id = ?)
		  AND (? = 0 OR b.user_id = ?)
		  AND (? = 0 OR a.username LIKE ? OR b.username LIKE ?)
		  AND (? = 0 OR b.created_at >= ?)
		  AND (? = 0 OR b.created_at < ?)
		  AND (? = 0 OR b.created_at <= ?)
		  AND (? = 0 OR b.token_name LIKE ?)
		  AND (? = 0 OR b.model LIKE ?)
		  AND (? = 0 OR b.type = ?)
`

type Record struct {
	Id              int64     `json:"id"`
	UserId          int64     `json:"user_id"`
	Username        string    `json:"username"`
	Type            string    `json:"type"`
	TokenName       string    `json:"token_name"`
	Model           string    `json:"model"`
	InputTokens     int64     `json:"input_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
	Quota           float64   `json:"quota"`
	Duration        float32   `json:"duration"`
	Detail          string    `json:"detail"`
	Prompts         string    `json:"prompts"`
	ResponsePrompts string    `json:"response_prompts"`
	Channel         int64     `json:"channel"`
	ChannelName     string    `json:"channel_name"`
	CreatedAt       time.Time `json:"created_at"`
}

type RecordQuery struct {
	UserId      int64  `json:"user_id"`
	Username    string `json:"username"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TokenName   string `json:"token_name"`
	Model       string `json:"model"`
	Type        string `json:"type"`
	ShowChannel bool   `json:"show_channel"`
	Self        bool   `json:"self"`
}

type RecordData struct {
	Total   int64    `json:"total"`
	Records []Record `json:"records"`
}

type RecordStats struct {
	BillingToday float32 `json:"billing_today"`
	BillingMonth float32 `json:"billing_month"`
	RequestToday int64   `json:"request_today"`
	RequestMonth int64   `json:"request_month"`
	Rpm          int64   `json:"rpm"`
	Tpm          int64   `json:"tpm"`
}

func recordStorageLocation() *time.Location {
	return channel.GetSystemTimeZoneLocation()
}

func parseRecordStorageTime(value []uint8) *time.Time {
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", string(value), recordStorageLocation())
	if err != nil {
		return nil
	}
	return &parsed
}

func CreateRecord(db *sql.DB, userId int64, username string, recordType string,
	tokenName string, model string, inputTokens int64, outputTokens int64,
	quota float64, duration float32, detail string, prompts string, responsePrompts string,
	channelId int, channelName string) {

	_, err := globals.ExecDb(db, `
			INSERT INTO billing (user_id, username, type, token_name, model, input_tokens, output_tokens, quota, duration, detail, prompts, response_prompts, channel, channel_name)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, userId, username, recordType, tokenName, model, inputTokens, outputTokens, quota, duration, detail, prompts, responsePrompts, channelId, channelName)
	if err != nil {
		globals.Warn(fmt.Sprintf("[billing] failed to create record: %s", err.Error()))
	}
}

func resolveChannelNameByModel(model string) string {
	if len(strings.TrimSpace(model)) == 0 || channel.ConduitInstance == nil {
		return ""
	}

	names := make([]string, 0)
	for _, item := range channel.ConduitInstance.GetActiveSequence() {
		if item != nil && item.IsHit(model) && !utils.Contains(item.GetName(), names) {
			names = append(names, item.GetName())
		}
	}

	return strings.Join(names, ", ")
}

func buildRecordTimeRange(value string, isEnd bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	type recordTimeLayout struct {
		layout      string
		hasTimeZone bool
	}
	layouts := []recordTimeLayout{
		{layout: "2006-01-02"},
		{layout: time.RFC3339Nano, hasTimeZone: true},
		{layout: time.RFC3339, hasTimeZone: true},
		{layout: "2006-01-02 15:04:05"},
		{layout: "2006-01-02T15:04:05"},
	}
	location := recordStorageLocation()

	for _, item := range layouts {
		parsed, err := time.ParseInLocation(item.layout, value, location)
		if err != nil {
			continue
		}
		if item.hasTimeZone {
			parsed = parsed.In(location)
		}

		if item.layout == "2006-01-02" {
			if isEnd {
				return parsed.AddDate(0, 0, 1).Format("2006-01-02 15:04:05"), nil
			}
			return parsed.Format("2006-01-02 15:04:05"), nil
		}

		return parsed.Format("2006-01-02 15:04:05"), nil
	}

	return "", fmt.Errorf("invalid time format: %s", value)
}

func sqlFlag(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

func recordLikeFilter(value string) string {
	return "%" + value + "%"
}

func buildRecordFilterArgs(isAdmin bool, userId int64, query RecordQuery) ([]interface{}, error) {
	userScopeEnabled := !isAdmin || query.Self
	adminUserEnabled := isAdmin && !query.Self && query.UserId > 0
	usernameEnabled := isAdmin && !query.Self && query.UserId <= 0 && query.Username != ""

	startEnabled := query.StartTime != ""
	startTime := ""
	if startEnabled {
		parsed, err := buildRecordTimeRange(query.StartTime, false)
		if err != nil {
			return nil, err
		}
		startTime = parsed
	}

	endExclusiveEnabled := false
	endInclusiveEnabled := false
	endTime := ""
	if query.EndTime != "" {
		parsed, err := buildRecordTimeRange(query.EndTime, true)
		if err != nil {
			return nil, err
		}
		endTime = parsed
		if len(strings.TrimSpace(query.EndTime)) == len("2006-01-02") {
			endExclusiveEnabled = true
		} else {
			endInclusiveEnabled = true
		}
	}

	tokenNameEnabled := query.TokenName != ""
	modelEnabled := query.Model != ""
	typeEnabled := query.Type != "" && query.Type != "all"
	usernameLike := recordLikeFilter(query.Username)

	return []interface{}{
		sqlFlag(userScopeEnabled), userId,
		sqlFlag(adminUserEnabled), query.UserId,
		sqlFlag(usernameEnabled), usernameLike, usernameLike,
		sqlFlag(startEnabled), startTime,
		sqlFlag(endExclusiveEnabled), endTime,
		sqlFlag(endInclusiveEnabled), endTime,
		sqlFlag(tokenNameEnabled), recordLikeFilter(query.TokenName),
		sqlFlag(modelEnabled), recordLikeFilter(query.Model),
		sqlFlag(typeEnabled), query.Type,
	}, nil
}

func GetUserRecordStats(db *sql.DB, userId int64) (RecordStats, error) {
	now := time.Now().In(recordStorageLocation())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var stats RecordStats
	err := globals.QueryRowDb(db, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'consume' AND created_at >= ? THEN quota ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'consume' AND created_at >= ? THEN quota ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'consume' AND created_at >= ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'consume' AND created_at >= ? THEN 1 ELSE 0 END), 0)
		FROM billing
		WHERE user_id = ?
	`,
		today.Format("2006-01-02 15:04:05"),
		month.Format("2006-01-02 15:04:05"),
		today.Format("2006-01-02 15:04:05"),
		month.Format("2006-01-02 15:04:05"),
		userId,
	).Scan(
		&stats.BillingToday,
		&stats.BillingMonth,
		&stats.RequestToday,
		&stats.RequestMonth,
	)
	if err != nil {
		return RecordStats{}, err
	}

	return stats, nil
}

func ListRecords(db *sql.DB, isAdmin bool, userId int64, page int64, query RecordQuery) (RecordData, error) {
	filterArgs, err := buildRecordFilterArgs(isAdmin, userId, query)
	if err != nil {
		return RecordData{}, err
	}

	var total int64
	countArgs := make([]interface{}, len(filterArgs))
	copy(countArgs, filterArgs)

	countQuery := `
		SELECT COUNT(*)
		FROM billing b
		LEFT JOIN auth a ON a.id = b.user_id
	` + recordFilterWhereSQL

	if err := globals.QueryRowDb(db, countQuery, countArgs...).Scan(&total); err != nil {
		return RecordData{}, err
	}

	queryArgs := append(filterArgs, pageSize, page*pageSize)
	rows, err := globals.QueryDb(db, `
		SELECT b.id, b.user_id, COALESCE(a.username, b.username, ''), b.type, COALESCE(b.token_name, ''),
		       b.model, b.input_tokens, b.output_tokens, b.quota, b.duration,
		       COALESCE(b.detail, ''), COALESCE(b.prompts, ''), COALESCE(b.response_prompts, ''),
		       COALESCE(b.channel, 0), COALESCE(b.channel_name, ''), b.created_at
		FROM billing b
		LEFT JOIN auth a ON a.id = b.user_id
	`+recordFilterWhereSQL+`
		ORDER BY b.id DESC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return RecordData{}, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var createdAt []uint8
		if err := rows.Scan(
			&r.Id, &r.UserId, &r.Username, &r.Type, &r.TokenName,
			&r.Model, &r.InputTokens, &r.OutputTokens, &r.Quota, &r.Duration,
			&r.Detail, &r.Prompts, &r.ResponsePrompts,
			&r.Channel, &r.ChannelName, &createdAt,
		); err != nil {
			return RecordData{}, err
		}
		if t := parseRecordStorageTime(createdAt); t != nil {
			r.CreatedAt = *t
		}
		if len(strings.TrimSpace(r.ChannelName)) == 0 {
			r.ChannelName = resolveChannelNameByModel(r.Model)
		}
		records = append(records, r)
	}

	if records == nil {
		records = []Record{}
	}

	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}

	return RecordData{Total: pages, Records: records}, nil
}
