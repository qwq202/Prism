package broadcast

type Broadcast struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
}

type Info struct {
	Index     int    `json:"index"`
	Content   string `json:"content"`
	Poster    string `json:"poster"`
	Type      string `json:"type"`
	StartAt   string `json:"start_at"`
	EndAt     string `json:"end_at"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type listResponse struct {
	Data []Info `json:"data"`
}

type createRequest struct {
	Content  string `json:"content"`
	Type     string `json:"type"`
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	IsActive *bool  `json:"is_active"`
}

type createResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

type updateRequest struct {
	Id       int    `json:"id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	IsActive *bool  `json:"is_active"`
}
