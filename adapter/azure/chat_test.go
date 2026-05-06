package azure

import "testing"

func TestGetChatResponseContentRejectsEmptyChoices(t *testing.T) {
	if _, err := getChatResponseContent(&ChatResponse{}); err == nil {
		t.Fatalf("expected empty choices to return an error")
	}
}
