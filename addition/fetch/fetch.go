package fetch

import (
	"bufio"
	"bytes"
	"chat/globals"
	"chat/utils"
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	ToolName         = "fetch_webpage"
	maxFetchBytes    = 2 * 1024 * 1024
	maxPageTextRunes = 12000
)

type pageResult struct {
	URL     string
	Title   string
	Content string
	Error   string
}

type ToolInput struct {
	URL      string `json:"url"`
	MaxChars *int   `json:"max_chars,omitempty"`
}

type ToolResult struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func BuildToolDefinition() *globals.FunctionTools {
	required := []string{"url"}

	tools := globals.FunctionTools{
		{
			Type: "function",
			Function: globals.ToolFunction{
				Name:        ToolName,
				Description: "Fetch the readable text content of a public http/https webpage. Use this when the user asks you to open, read, summarize, analyze, or answer questions about a URL.",
				Parameters: globals.ToolParameters{
					Type: "object",
					Properties: globals.ToolProperties{
						"url": {
							"type":        "string",
							"description": "The public http or https URL to fetch.",
						},
						"max_chars": {
							"type":        "integer",
							"description": "Optional maximum number of characters to return. Defaults to 12000 and is capped by the application.",
						},
					},
					Required: &required,
				},
			},
		},
	}

	return &tools
}

func toolResultMessage(callID string, result ToolResult) globals.Message {
	return globals.Message{
		Role:       globals.Tool,
		Content:    utils.Marshal(result),
		ToolCallId: utils.ToPtr(callID),
	}
}

func ExecuteToolCall(call globals.ToolCall) globals.Message {
	result := ToolResult{
		Status: "error",
		Action: call.Function.Name,
	}

	if call.Function.Name != ToolName {
		result.Error = "unsupported tool"
		return toolResultMessage(call.Id, result)
	}

	input, err := utils.UnmarshalString[ToolInput](call.Function.Arguments)
	if err != nil {
		result.Error = "invalid tool arguments"
		return toolResultMessage(call.Id, result)
	}

	input.URL = strings.TrimSpace(input.URL)
	if input.URL == "" {
		result.Error = "url is required"
		return toolResultMessage(call.Id, result)
	}

	limit := maxPageTextRunes
	if input.MaxChars != nil && *input.MaxChars > 0 && *input.MaxChars < limit {
		limit = *input.MaxChars
	}

	page := fetchURL(input.URL, limit)
	result.URL = page.URL
	result.Title = page.Title
	result.Content = page.Content
	if page.Error != "" {
		result.Error = page.Error
		return toolResultMessage(call.Id, result)
	}

	result.Status = "success"
	result.Message = "webpage fetched"
	return toolResultMessage(call.Id, result)
}

func ExecuteToolCalls(calls *globals.ToolCalls) []globals.Message {
	if calls == nil || len(*calls) == 0 {
		return nil
	}

	messages := make([]globals.Message, 0, len(*calls))
	for _, call := range *calls {
		messages = append(messages, ExecuteToolCall(call))
	}

	return messages
}

func fetchURL(rawURL string, maxTextRunes int) pageResult {
	result := pageResult{URL: rawURL}
	if err := validateURL(rawURL); err != nil {
		result.Error = err.Error()
		return result
	}

	client := &http.Client{
		Timeout:   12 * time.Second,
		Transport: secureTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return validateURL(req.URL.String())
		},
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.5")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if len(body) > maxFetchBytes {
		result.Error = "page is too large to fetch safely"
		return result
	}

	contentType := resp.Header.Get("Content-Type")
	utf8Body, err := decodeBody(body, contentType)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	if strings.Contains(strings.ToLower(contentType), "html") || looksLikeHTML(utf8Body) {
		result.Title, result.Content = htmlToText(utf8Body)
	} else {
		result.Content = normalizeSpace(utf8Body)
	}
	result.Content = limitRunes(result.Content, maxTextRunes)
	if strings.TrimSpace(result.Content) == "" {
		result.Error = "no readable text content found"
	}

	return result
}

func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are supported")
	}
	if parsed.User != nil {
		return fmt.Errorf("URL credentials are not allowed")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL host is empty")
	}
	if isBlockedHost(host) {
		return fmt.Errorf("local or private network URLs are not allowed")
	}

	return nil
}

func secureTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			if isBlockedHost(host) {
				return nil, fmt.Errorf("local or private network URLs are not allowed")
			}

			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}

			if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok && isBlockedIP(tcpAddr.IP) {
				_ = conn.Close()
				return nil, fmt.Errorf("local or private network URLs are not allowed")
			}

			return conn, nil
		},
	}
}

func isBlockedHost(host string) bool {
	host = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(host), "."))
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.Contains(host, "%") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return isBlockedIP(ip)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return true
		}
	}
	return false
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return true
	}

	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

func decodeBody(body []byte, contentType string) (string, error) {
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return "", err
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func looksLikeHTML(content string) bool {
	content = strings.ToLower(content)
	return strings.Contains(content, "<html") ||
		strings.Contains(content, "<body") ||
		strings.Contains(content, "<!doctype html")
}

func htmlToText(content string) (string, string) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(content))
	var text strings.Builder
	var title strings.Builder
	stack := make([]string, 0)

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case xhtml.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return normalizeSpace(title.String()), normalizeSpace(text.String())
			}
			return normalizeSpace(title.String()), normalizeSpace(text.String())
		case xhtml.StartTagToken:
			name, _ := tokenizer.TagName()
			stack = append(stack, string(name))
		case xhtml.EndTagToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xhtml.TextToken:
			if shouldSkipText(stack) {
				continue
			}
			value := html.UnescapeString(string(tokenizer.Text()))
			if len(stack) > 0 && stack[len(stack)-1] == "title" {
				title.WriteString(value)
				title.WriteByte(' ')
				continue
			}
			text.WriteString(value)
			text.WriteByte(' ')
		}
	}
}

func shouldSkipText(stack []string) bool {
	for _, tag := range stack {
		switch tag {
		case "script", "style", "noscript", "svg", "canvas":
			return true
		}
	}
	return false
}

func normalizeSpace(content string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 1024), maxFetchBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return ' '
			}
			return r
		}, scanner.Text()))
		if line == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line)
	}
	return builder.String()
}

func limitRunes(content string, limit int) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "\n...[truncated]"
}
