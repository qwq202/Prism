package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"chat/globals"
	"chat/utils"

	"github.com/gin-gonic/gin"
)

type TavilyResponse struct {
	Query        string `json:"query"`
	ResponseTime any    `json:"response_time"`
	Results      []struct {
		Url             string  `json:"url"`
		Title           string  `json:"title"`
		Content         string  `json:"content"`
		Score           float64 `json:"score"`
		Favicon         *string `json:"favicon,omitempty"`
		PublishedDate   *string `json:"published_date,omitempty"`
		RawContent      *string `json:"raw_content,omitempty"`
		ContentSource   *string `json:"content_source,omitempty"`
		ResponseContent *string `json:"response_content,omitempty"`
	} `json:"results"`
}

type TavilyUsageResponse struct {
	Key struct {
		Usage         float64 `json:"usage"`
		Limit         float64 `json:"limit"`
		SearchUsage   float64 `json:"search_usage"`
		ExtractUsage  float64 `json:"extract_usage"`
		CrawlUsage    float64 `json:"crawl_usage"`
		MapUsage      float64 `json:"map_usage"`
		ResearchUsage float64 `json:"research_usage"`
	} `json:"key"`
	Account struct {
		CurrentPlan   string  `json:"current_plan"`
		PlanUsage     float64 `json:"plan_usage"`
		PlanLimit     float64 `json:"plan_limit"`
		PaygoUsage    float64 `json:"paygo_usage"`
		PaygoLimit    float64 `json:"paygo_limit"`
		SearchUsage   float64 `json:"search_usage"`
		ExtractUsage  float64 `json:"extract_usage"`
		CrawlUsage    float64 `json:"crawl_usage"`
		MapUsage      float64 `json:"map_usage"`
		ResearchUsage float64 `json:"research_usage"`
	} `json:"account"`
}

type TavilyUsageView struct {
	Usage         float64 `json:"usage"`
	Limit         float64 `json:"limit"`
	Remaining     float64 `json:"remaining"`
	Percent       float64 `json:"percent"`
	SearchUsage   float64 `json:"search_usage"`
	ExtractUsage  float64 `json:"extract_usage"`
	CrawlUsage    float64 `json:"crawl_usage"`
	MapUsage      float64 `json:"map_usage"`
	ResearchUsage float64 `json:"research_usage"`
	CurrentPlan   string  `json:"current_plan,omitempty"`
}

type TavilyUsageForm struct {
	ApiKey string `json:"api_key"`
}

func formatResponse(data *TavilyResponse) string {
	res := make([]string, 0)
	for _, item := range data.Results {
		if item.Content == "" || item.Url == "" || item.Title == "" {
			continue
		}

		res = append(res, fmt.Sprintf("%s (%s): %s", item.Title, item.Url, item.Content))
	}

	return strings.Join(res, "\n")
}

func tavilyUsageError(status string, body []byte) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, key := range []string{"error", "message", "detail"} {
			if value, ok := payload[key]; ok {
				text := strings.TrimSpace(utils.ToString(value))
				if text != "" {
					return fmt.Errorf("tavily usage request failed (%s): %s", status, text)
				}
			}
		}
	}

	text := strings.TrimSpace(string(body))
	if text != "" {
		return fmt.Errorf("tavily usage request failed (%s): %s", status, utils.Extract(text, 200, "..."))
	}

	return fmt.Errorf("tavily usage request failed: %s", status)
}

func fetchTavilyUsage(apiKey string) (*TavilyUsageResponse, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("tavily api key is empty")
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.tavily.com/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, tavilyUsageError(resp.Status, body)
	}

	var usage TavilyUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, err
	}

	return &usage, nil
}

func toTavilyUsageView(usage *TavilyUsageResponse) TavilyUsageView {
	limit := usage.Key.Limit
	used := usage.Key.Usage
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	percent := 0.0
	if limit > 0 {
		percent = remaining / limit * 100
	}

	return TavilyUsageView{
		Usage:         used,
		Limit:         limit,
		Remaining:     remaining,
		Percent:       percent,
		SearchUsage:   usage.Key.SearchUsage,
		ExtractUsage:  usage.Key.ExtractUsage,
		CrawlUsage:    usage.Key.CrawlUsage,
		MapUsage:      usage.Key.MapUsage,
		ResearchUsage: usage.Key.ResearchUsage,
		CurrentPlan:   usage.Account.CurrentPlan,
	}
}

func createTavilyRequest(query string) (*TavilyResponse, error) {
	data, err := utils.Post("https://api.tavily.com/search", map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", globals.SearchApiKey),
	}, map[string]interface{}{
		"query":        query,
		"topic":        globals.SearchTopic,
		"search_depth": globals.SearchDepth,
		"max_results":  globals.SearchMaxResults,
	})

	if err != nil {
		return nil, err
	}

	return utils.MapToRawStruct[TavilyResponse](data)
}

func GenerateSearchResult(q string) (string, error) {
	if strings.TrimSpace(globals.SearchApiKey) == "" {
		return "search failed: tavily api key is empty", errors.New("search failed: tavily api key is empty")
	}

	res, err := createTavilyRequest(q)
	if err != nil {
		globals.Warn(fmt.Sprintf("[web] failed to get search result: %s (query: %s)", err.Error(), utils.Extract(q, 20, "...")))

		content := fmt.Sprintf("search failed: %s", err.Error())
		return content, errors.New(content)
	}

	content := formatResponse(res)
	globals.Debug(fmt.Sprintf("[web] search result: %s (query: %s)", utils.Extract(content, 50, "..."), utils.Extract(q, 20, "...")))

	if globals.SearchCrop {
		globals.Debug(fmt.Sprintf("[web] crop search result length %d to %d max", len(content), globals.SearchCropLength))
		return utils.Extract(content, globals.SearchCropLength, "..."), nil
	}
	return content, nil
}

func TestSearch(c *gin.Context) {
	// get `query` param from query
	query := c.Query("query")

	res, err := GenerateSearchResult(query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status": true,
			"result": res,
		})
	}
}

func TestTavilyUsage(c *gin.Context) {
	var form TavilyUsageForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	usage, err := fetchTavilyUsage(form.ApiKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   toTavilyUsageView(usage),
	})
}
