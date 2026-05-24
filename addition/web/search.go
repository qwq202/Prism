package web

import (
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
