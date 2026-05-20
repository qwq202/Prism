package manager

type VideoJobError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type VideoJob struct {
	CompletedAt        *int64         `json:"completed_at,omitempty"`
	CreatedAt          int64          `json:"created_at"`
	Error              *VideoJobError `json:"error,omitempty"`
	ExpiresAt          *int64         `json:"expires_at,omitempty"`
	Id                 string         `json:"id"`
	Model              string         `json:"model"`
	Object             string         `json:"object"`
	Progress           *int           `json:"progress,omitempty"`
	Prompt             string         `json:"prompt"`
	RemixedFromVideoId *string        `json:"remixed_from_video_id,omitempty"`
	Seconds            string         `json:"seconds"`
	Size               string         `json:"size"`
	Status             string         `json:"status"`
}
