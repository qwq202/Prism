package auth

import "testing"

func TestDecodeDeeptrainResponseRejectsMalformedPayloads(t *testing.T) {
	tests := []struct {
		name string
		res  interface{}
	}{
		{name: "nil", res: nil},
		{name: "array", res: []interface{}{map[string]interface{}{"status": true}}},
		{name: "string status", res: map[string]interface{}{"status": "true"}},
		{name: "missing status", res: map[string]interface{}{"balance": 10}},
		{name: "false status", res: map[string]interface{}{"status": false, "balance": 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := decodeDeeptrainResponse[BalanceResponse](tt.res); ok {
				t.Fatalf("expected malformed deeptrain response to be rejected")
			}
		})
	}
}

func TestDecodeDeeptrainResponseDecodesSuccessfulPayload(t *testing.T) {
	resp, ok := decodeDeeptrainResponse[BalanceResponse](map[string]interface{}{
		"status":  true,
		"balance": 12.5,
	})
	if !ok {
		t.Fatalf("expected deeptrain response to decode")
	}
	if resp.Balance != 12.5 {
		t.Fatalf("expected balance 12.5, got %f", resp.Balance)
	}
}
