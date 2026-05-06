package analysis

type ModelData struct {
	Model string  `json:"model"`
	Data  []int64 `json:"data"`
}

type ModelChartForm struct {
	Date  []string    `json:"date"`
	Value []ModelData `json:"value"`
}

type RequestChartForm struct {
	Date  []string `json:"date"`
	Value []int64  `json:"value"`
}

type BillingChartForm struct {
	Date  []string  `json:"date"`
	Value []float32 `json:"value"`
}

type ErrorChartForm struct {
	Date  []string `json:"date"`
	Value []int64  `json:"value"`
}

type ChannelStats struct {
	ChannelId int     `json:"channel_id"`
	Requests  int64   `json:"requests"`
	Errors    int64   `json:"errors"`
	ErrorRate float64 `json:"error_rate"`
}

type ChannelStatsResponse struct {
	Stats []ChannelStats `json:"stats"`
}

type ActiveUserChartForm struct {
	Date  []string `json:"date"`
	Value []int64  `json:"value"`
}

type RegistrationChartForm struct {
	Date  []string `json:"date"`
	Value []int64  `json:"value"`
}

type ConversionFunnelForm struct {
	Registered       int64 `json:"registered"`
	EverSubscribed   int64 `json:"ever_subscribed"`
	ActiveSubscribed int64 `json:"active_subscribed"`
}
