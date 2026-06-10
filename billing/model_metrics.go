package billing

import (
	"chat/adapter"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	ModelMetricErrorAvailability = "availability_error"
	ModelMetricErrorRequest      = "request_error"
	ModelMetricErrorEmpty        = "empty_response"
)

type ModelUsageTrendPoint struct {
	Time         string  `json:"time"`
	Requests     int64   `json:"requests"`
	Successes    int64   `json:"successes"`
	AvgLatency   float64 `json:"avg_latency"`
	Availability float64 `json:"availability"`
}

type ModelUsageStats struct {
	Model                string                 `json:"model"`
	WindowHours          int                    `json:"window_hours"`
	RequestCount         int64                  `json:"request_count"`
	SuccessCount         int64                  `json:"success_count"`
	FailureCount         int64                  `json:"failure_count"`
	AvailabilityFailures int64                  `json:"availability_failures"`
	TPS                  float64                `json:"tps"`
	AvgLatency           float64                `json:"avg_latency"`
	SuccessRate          float64                `json:"success_rate"`
	Availability         float64                `json:"availability"`
	LatencyTrend         []ModelUsageTrendPoint `json:"latency_trend"`
	AvailabilityTrend    []ModelUsageTrendPoint `json:"availability_trend"`
}

type modelUsageBucket struct {
	requests             int64
	successes            int64
	availabilityFailures int64
	outputTokens         int64
	duration             float64
	latencyTotal         float64
	latencyCount         int64
}

func truncateMetricError(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	if len(message) <= 500 {
		return message
	}

	return message[:500]
}

func modelMetricErrorType(buffer *utils.Buffer, err error, success bool) string {
	if success {
		return ""
	}
	if adapter.IsAvailableError(err) {
		return ModelMetricErrorAvailability
	}
	if err != nil {
		return ModelMetricErrorRequest
	}
	if buffer == nil || buffer.IsEmpty() {
		return ModelMetricErrorEmpty
	}

	return ModelMetricErrorRequest
}

func RecordModelUsageMetric(db *sql.DB, model string, buffer *utils.Buffer, err error) {
	model = strings.TrimSpace(model)
	if db == nil || model == "" {
		return
	}

	var inputTokens int64
	var outputTokens int64
	var duration float32
	if buffer != nil {
		inputTokens = int64(buffer.CountRecordInputToken())
		outputTokens = int64(buffer.CountRecordOutputToken())
		duration = buffer.GetDuration()
	}

	success := !adapter.IsAvailableError(err) && buffer != nil && !buffer.IsEmpty()
	errorType := modelMetricErrorType(buffer, err, success)
	createdAt := time.Now().In(recordStorageLocation()).Format("2006-01-02 15:04:05")

	_, execErr := globals.ExecDb(db, `
		INSERT INTO model_usage_metrics (
			model, success, error_type, input_tokens, output_tokens,
			duration, error, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, model, success, errorType, inputTokens, outputTokens, duration, truncateMetricError(err), createdAt)
	if execErr != nil {
		globals.Warn(fmt.Sprintf("[metrics] failed to record model usage: %s", execErr.Error()))
	}
}

func parseModelMetricTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	location := recordStorageLocation()
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return parsed.In(location), true
		}
	}

	return time.Time{}, false
}

func ratio(part int64, total int64) float64 {
	if total <= 0 {
		return 0
	}

	return float64(part) / float64(total)
}

func buildModelUsagePoint(bucketStart time.Time, bucket modelUsageBucket) ModelUsageTrendPoint {
	avgLatency := 0.0
	if bucket.latencyCount > 0 {
		avgLatency = bucket.latencyTotal / float64(bucket.latencyCount)
	}

	return ModelUsageTrendPoint{
		Time:         bucketStart.Format(time.RFC3339),
		Requests:     bucket.requests,
		Successes:    bucket.successes,
		AvgLatency:   avgLatency,
		Availability: ratio(bucket.requests-bucket.availabilityFailures, bucket.requests),
	}
}

func GetModelUsageStats(db *sql.DB, model string, windowHours int) (ModelUsageStats, error) {
	model = strings.TrimSpace(model)
	if windowHours <= 0 {
		windowHours = 24
	}

	stats := ModelUsageStats{
		Model:             model,
		WindowHours:       windowHours,
		LatencyTrend:      make([]ModelUsageTrendPoint, 0, windowHours),
		AvailabilityTrend: make([]ModelUsageTrendPoint, 0, windowHours),
	}
	if db == nil || model == "" {
		return stats, nil
	}

	location := recordStorageLocation()
	end := time.Now().In(location).Truncate(time.Hour).Add(time.Hour)
	start := end.Add(-time.Duration(windowHours) * time.Hour)
	buckets := make([]modelUsageBucket, windowHours)

	rows, err := globals.QueryDb(db, `
		SELECT CASE WHEN success THEN 1 ELSE 0 END, COALESCE(error_type, ''),
		       input_tokens, output_tokens, duration, created_at
		FROM model_usage_metrics
		WHERE model = ? AND created_at >= ? AND created_at < ?
		ORDER BY created_at ASC
	`, model, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var successValue int
		var errorType string
		var inputTokens int64
		var outputTokens int64
		var duration float64
		var createdAtRaw string
		if err := rows.Scan(&successValue, &errorType, &inputTokens, &outputTokens, &duration, &createdAtRaw); err != nil {
			return stats, err
		}

		createdAt, ok := parseModelMetricTime(createdAtRaw)
		if !ok || createdAt.Before(start) || !createdAt.Before(end) {
			continue
		}

		index := int(createdAt.Sub(start) / time.Hour)
		if index < 0 || index >= windowHours {
			continue
		}

		bucket := &buckets[index]
		success := successValue == 1
		bucket.requests++
		stats.RequestCount++
		if success {
			bucket.successes++
			stats.SuccessCount++
			bucket.outputTokens += outputTokens
			if duration > 0 {
				bucket.duration += duration
				bucket.latencyTotal += duration
				bucket.latencyCount++
			}
		} else {
			stats.FailureCount++
		}
		if errorType == ModelMetricErrorAvailability {
			bucket.availabilityFailures++
			stats.AvailabilityFailures++
		}
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}

	var latencyTotal float64
	var latencyCount int64
	var outputTokens int64
	var durationTotal float64
	for i, bucket := range buckets {
		latencyTotal += bucket.latencyTotal
		latencyCount += bucket.latencyCount
		outputTokens += bucket.outputTokens
		durationTotal += bucket.duration

		point := buildModelUsagePoint(start.Add(time.Duration(i)*time.Hour), bucket)
		stats.LatencyTrend = append(stats.LatencyTrend, point)
		stats.AvailabilityTrend = append(stats.AvailabilityTrend, point)
	}

	if latencyCount > 0 {
		stats.AvgLatency = latencyTotal / float64(latencyCount)
	}
	if durationTotal > 0 {
		stats.TPS = float64(outputTokens) / durationTotal
	}
	stats.SuccessRate = ratio(stats.SuccessCount, stats.RequestCount)
	stats.Availability = ratio(stats.RequestCount-stats.AvailabilityFailures, stats.RequestCount)

	return stats, nil
}
