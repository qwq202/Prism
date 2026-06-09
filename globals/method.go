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

func (m *ClaudeHiddenMetadata) IsEmpty() bool {
	return m == nil || len(m.ThinkingBlocks) == 0
}

func NormalizeClaudeThinkingBlocks(blocks []ClaudeThinkingBlock, limit int) []ClaudeThinkingBlock {
	if limit <= 0 || limit > ClaudeThinkingBlockLimit {
		limit = ClaudeThinkingBlockLimit
	}

	result := make([]ClaudeThinkingBlock, 0, limit)
	for _, block := range blocks {
		thinking := strings.TrimSpace(block.Thinking)
		signature := strings.TrimSpace(block.Signature)

		if len(thinking) == 0 && len(signature) == 0 {
			continue
		}

		if len(thinking) > ClaudeThinkingTextMaxBytes {
			thinking = thinking[:ClaudeThinkingTextMaxBytes]
		}

		if len(signature) > ClaudeThinkingSignatureMaxBytes {
			signature = signature[:ClaudeThinkingSignatureMaxBytes]
		}

		result = append(result, ClaudeThinkingBlock{
			Thinking:  thinking,
			Signature: signature,
		})
		if len(result) >= limit {
			break
		}
	}

	return result
}

func (m *ClaudeHiddenMetadata) Normalized(limit int) *ClaudeHiddenMetadata {
	if m == nil {
		return nil
	}

	blocks := NormalizeClaudeThinkingBlocks(m.ThinkingBlocks, limit)
	if len(blocks) == 0 {
		return nil
	}

	return &ClaudeHiddenMetadata{
		ThinkingBlocks: blocks,
	}
}

func MergeClaudeHiddenMetadata(limit int, metadata ...*ClaudeHiddenMetadata) *ClaudeHiddenMetadata {
	merged := make([]ClaudeThinkingBlock, 0, ClaudeThinkingBlockLimit)
	for _, item := range metadata {
		if item == nil {
			continue
		}

		merged = append(merged, item.ThinkingBlocks...)
	}

	return (&ClaudeHiddenMetadata{
		ThinkingBlocks: merged,
	}).Normalized(limit)
}

func (s *BuiltinToolUsageStatus) IsEmpty() bool {
	return s == nil || (!s.Enabled && !s.Sent && !s.Used)
}

func (u *BuiltinToolUsage) IsEmpty() bool {
	return u == nil || u.CodeExecution.IsEmpty()
}

// Chunk-level emptiness controls whether a stream delta should be emitted.
// Hidden metadata is intentionally considered non-empty so metadata deltas can be forwarded.
func (c *Chunk) IsEmpty() bool {
	return len(c.Content) == 0 &&
		c.ToolCall == nil &&
		c.FunctionCall == nil &&
		c.ReasoningContent == nil &&
		c.GeminiHiddenMetadata.IsEmpty() &&
		c.ClaudeHiddenMetadata.IsEmpty() &&
		c.BuiltinToolUsage.IsEmpty()
}
