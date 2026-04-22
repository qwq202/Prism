package globals

import "strings"

func (m *GeminiHiddenMetadata) IsEmpty() bool {
	return m == nil || len(m.ThoughtSignatures) == 0
}

func NormalizeGeminiThoughtSignatures(signatures []string, limit int) []string {
	if limit <= 0 || limit > GeminiThoughtSignatureLimit {
		limit = GeminiThoughtSignatureLimit
	}

	result := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)

	for _, signature := range signatures {
		normalized := strings.TrimSpace(signature)
		if len(normalized) == 0 || len(normalized) > GeminiThoughtSignatureMaxBytes {
			continue
		}

		if _, hit := seen[normalized]; hit {
			continue
		}

		result = append(result, normalized)
		seen[normalized] = struct{}{}
		if len(result) >= limit {
			break
		}
	}

	return result
}

func (m *GeminiHiddenMetadata) Normalized(limit int) *GeminiHiddenMetadata {
	if m == nil {
		return nil
	}

	signatures := NormalizeGeminiThoughtSignatures(m.ThoughtSignatures, limit)
	if len(signatures) == 0 {
		return nil
	}

	return &GeminiHiddenMetadata{
		ThoughtSignatures: signatures,
	}
}

func MergeGeminiHiddenMetadata(limit int, metadata ...*GeminiHiddenMetadata) *GeminiHiddenMetadata {
	merged := make([]string, 0, GeminiThoughtSignatureLimit)
	for _, item := range metadata {
		if item == nil {
			continue
		}

		merged = append(merged, item.ThoughtSignatures...)
	}

	return (&GeminiHiddenMetadata{
		ThoughtSignatures: merged,
	}).Normalized(limit)
}

// Chunk-level emptiness controls whether a stream delta should be emitted.
// Hidden metadata is intentionally considered non-empty so metadata deltas can be forwarded.
func (c *Chunk) IsEmpty() bool {
	return len(c.Content) == 0 && c.ToolCall == nil && c.FunctionCall == nil && c.ReasoningContent == nil && c.GeminiHiddenMetadata.IsEmpty()
}
