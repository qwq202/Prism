package billing

import (
	"chat/connection"
	"chat/globals"
	"database/sql"
	"math"
	"testing"
	"time"
)

func insertModelUsageMetric(
	t *testing.T,
	db *sql.DB,
	model string,
	success bool,
	errorType string,
	inputTokens int64,
	outputTokens int64,
	duration float64,
	createdAt time.Time,
) {
	t.Helper()

	if _, err := globals.ExecDb(db, `
		INSERT INTO model_usage_metrics (
			model, success, error_type, input_tokens, output_tokens,
			duration, error, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, model, success, errorType, inputTokens, outputTokens, duration, "", createdAt.In(recordStorageLocation()).Format("2006-01-02 15:04:05")); err != nil {
		t.Fatalf("insert model usage metric: %v", err)
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("expected %s %.4f, got %.4f", name, want, got)
	}
}

func TestGetModelUsageStatsAggregatesRecentMetrics(t *testing.T) {
	db := openBillingTestDB(t)
	connection.CreateModelUsageMetricsTable(db)

	now := time.Now().In(recordStorageLocation()).Add(-30 * time.Minute)
	insertModelUsageMetric(t, db, "deepseek-v4-flash", true, "", 100, 20, 2, now)
	insertModelUsageMetric(t, db, "deepseek-v4-flash", true, "", 80, 30, 3, now.Add(-10*time.Minute))
	insertModelUsageMetric(t, db, "deepseek-v4-flash", false, ModelMetricErrorAvailability, 0, 0, 1, now.Add(-20*time.Minute))
	insertModelUsageMetric(t, db, "gpt-5.5", true, "", 10, 10, 1, now)
	insertModelUsageMetric(t, db, "deepseek-v4-flash", true, "", 10, 1000, 1, now.Add(-25*time.Hour))

	stats, err := GetModelUsageStats(db, "deepseek-v4-flash", 24)
	if err != nil {
		t.Fatalf("get model usage stats: %v", err)
	}

	if stats.RequestCount != 3 || stats.SuccessCount != 2 || stats.FailureCount != 1 {
		t.Fatalf("unexpected counters: %#v", stats)
	}
	if stats.AvailabilityFailures != 1 {
		t.Fatalf("expected one availability failure, got %d", stats.AvailabilityFailures)
	}
	assertClose(t, "avg latency", stats.AvgLatency, 2.5)
	assertClose(t, "tps", stats.TPS, 10)
	assertClose(t, "success rate", stats.SuccessRate, 2.0/3.0)
	assertClose(t, "availability", stats.Availability, 2.0/3.0)

	if len(stats.LatencyTrend) != 24 || len(stats.AvailabilityTrend) != 24 {
		t.Fatalf("expected 24 hourly trend points, got latency=%d availability=%d", len(stats.LatencyTrend), len(stats.AvailabilityTrend))
	}
}

func TestGetModelUsageStatsSeparatesAvailabilityFromOtherFailures(t *testing.T) {
	db := openBillingTestDB(t)
	connection.CreateModelUsageMetricsTable(db)

	now := time.Now().In(recordStorageLocation()).Add(-15 * time.Minute)
	insertModelUsageMetric(t, db, "mimo-v2.5-pro", false, ModelMetricErrorEmpty, 0, 0, 0.4, now)

	stats, err := GetModelUsageStats(db, "mimo-v2.5-pro", 24)
	if err != nil {
		t.Fatalf("get model usage stats: %v", err)
	}

	if stats.RequestCount != 1 || stats.SuccessCount != 0 || stats.FailureCount != 1 {
		t.Fatalf("unexpected counters: %#v", stats)
	}
	assertClose(t, "success rate", stats.SuccessRate, 0)
	assertClose(t, "availability", stats.Availability, 1)
}
