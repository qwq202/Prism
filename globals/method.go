package globals

func (m *GeminiHiddenMetadata) IsEmpty() bool {
	return m == nil || m.ThoughtSignature == nil || len(*m.ThoughtSignature) == 0
}

func (c *Chunk) IsEmpty() bool {
	return len(c.Content) == 0 && c.ToolCall == nil && c.FunctionCall == nil && c.GeminiHiddenMetadata.IsEmpty()
}
