package utils

import (
	"runtime"
	"strings"
	"testing"
)

func BenchmarkBufferStreamingWrite(b *testing.B) {
	chunk := strings.Repeat("x", 32)

	b.ReportAllocs()
	b.SetBytes(int64(len(chunk) * 4096))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := &Buffer{}
		for j := 0; j < 4096; j++ {
			buffer.Write(chunk)
		}
		runtime.KeepAlive(buffer.Read())
	}
}
