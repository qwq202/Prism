package manager

import (
	"chat/globals"
	"chat/utils"
	"strings"
	"testing"
)

func BenchmarkRealtimeQuotaProjection(b *testing.B) {
	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	buffer.Write(strings.Repeat("streamed response content ", 4096))
	chunk := &globals.Chunk{Content: "next chunk"}
	limiter := realtimeQuotaLimiter{enabled: true, limit: 1e30}

	b.ReportAllocs()
	b.SetBytes(int64(len(buffer.Read())))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !limiter.allowsProjectedChunk(buffer, chunk) {
			b.Fatal("unexpected quota rejection")
		}
	}
}
