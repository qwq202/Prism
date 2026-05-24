package utils

import (
	"bytes"
	"chat/globals"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-json"
	"golang.org/x/net/proxy"
)

const (
	maxHTTPLogBodyBytes   = 64 * 1024
	maxHTTPErrorBodyBytes = 1024 * 1024
)

func newClient(c []globals.ProxyConfig) *http.Client {
	client := &http.Client{
		Timeout: globals.HttpMaxTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	if len(c) == 0 {
		return client
	}

	config := c[0]
	if config.ProxyType == globals.NoneProxyType {
		return client
	}

	if config.ProxyType == globals.HttpProxyType || config.ProxyType == globals.HttpsProxyType {
		proxyUrl, err := url.Parse(config.Proxy)
		if len(config.Username) > 0 || len(config.Password) > 0 {
			proxyUrl.User = url.UserPassword(config.Username, config.Password)
		}

		if err != nil {
			globals.Warn(fmt.Sprintf("failed to parse proxy url: %s", err))
			return client
		}
		client.Transport = &http.Transport{
			Proxy:           http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else if config.ProxyType == globals.Socks5ProxyType {
		var auth *proxy.Auth
		if len(config.Username) > 0 || len(config.Password) > 0 {
			auth = &proxy.Auth{
				User:     config.Username,
				Password: config.Password,
			}
		}

		dialer, err := proxy.SOCKS5("tcp", config.Proxy, auth, proxy.Direct)
		if err != nil {
			globals.Warn(fmt.Sprintf("failed to create socks5 proxy: %s", err))
			return client
		}

		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}

		client.Transport = &http.Transport{
			DialContext:     dialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	globals.Debug(fmt.Sprintf("[proxy] configured proxy: %s", config.Proxy))
	return client
}

func truncateLogText(data []byte) (string, bool) {
	if len(data) <= maxHTTPLogBodyBytes {
		return string(data), false
	}

	end := maxHTTPLogBodyBytes
	for end > 0 && !utf8.Valid(data[:end]) {
		end--
	}
	if end == 0 {
		end = maxHTTPLogBodyBytes
	}

	return string(data[:end]), true
}

func readErrorBody(body io.Reader) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxHTTPErrorBodyBytes+1))
	if err != nil {
		return data, false, err
	}
	if len(data) <= maxHTTPErrorBodyBytes {
		return data, false, nil
	}
	return data[:maxHTTPErrorBodyBytes], true, nil
}

func appendTruncatedNotice(content string, truncated bool, limit int) string {
	if !truncated {
		return content
	}
	return fmt.Sprintf("%s\n...[truncated to %s]", content, formatSize(limit))
}

func fillHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

func formatBodyForLog(data []byte, contentType string) string {
	if len(data) == 0 {
		return ""
	}

	isBinary := false
	if contentType != "" {
		contentType = strings.ToLower(contentType)
		binaryTypes := []string{
			"video/", "image/", "audio/",
			"application/octet-stream",
			"application/pdf",
			"application/zip",
			"application/x-",
		}
		for _, bt := range binaryTypes {
			if strings.HasPrefix(contentType, bt) {
				isBinary = true
				break
			}
		}
	}

	if !isBinary {
		if !utf8.Valid(data) {
			isBinary = true
		} else {
			nonPrintableCount := 0
			for _, b := range data {
				if b < 32 && b != 9 && b != 10 && b != 13 {
					nonPrintableCount++
				}
			}
			if len(data) > 0 && float64(nonPrintableCount)/float64(len(data)) > 0.05 {
				isBinary = true
			}
		}
	}

	if isBinary {
		detectedType := contentType
		if detectedType == "" {
			detectedType = http.DetectContentType(data)
		}
		size := len(data)
		sizeStr := formatSize(size)
		return fmt.Sprintf("[Binary Content] Type: %s, Size: %s (%d bytes)", detectedType, sizeStr, size)
	}

	content, truncated := truncateLogText(data)
	return appendTruncatedNotice(content, truncated, maxHTTPLogBodyBytes)
}

func formatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func Http(uri string, method string, ptr interface{}, headers map[string]string, body io.Reader, config []globals.ProxyConfig) (err error) {
	var requestBody io.Reader = body
	formattedRequestBody := ""
	if globals.DebugMode {
		if body != nil {
			if data, readErr := io.ReadAll(body); readErr == nil {
				formattedRequestBody = formatBodyForLog(data, "")
				requestBody = bytes.NewReader(data)
			} else {
				formattedRequestBody = fmt.Sprintf("[Body Read Error] %s", readErr)
			}
		}
		globals.Debug(fmt.Sprintf("[http] %s %s\nheaders: \n%s\nbody: \n%s", method, uri, Marshal(headers), formattedRequestBody))
	}

	req, err := http.NewRequest(method, uri, requestBody)
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http] failed to create request: %s", err))
		}

		return err
	}
	fillHeaders(req, headers)

	client := newClient(config)
	resp, err := client.Do(req)
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http] failed to send request: %s", err))
		}

		return err
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		if globals.DebugMode {
			contentType := resp.Header.Get("Content-Type")
			formattedBody := formatBodyForLog(respData, contentType)
			globals.Debug(fmt.Sprintf("[http] failed to read response: %s\nresponse: %s", err, formattedBody))
		}
		return err
	}

	contentType := resp.Header.Get("Content-Type")

	if globals.DebugMode {
		formattedBody := formatBodyForLog(respData, contentType)
		globals.Debug(fmt.Sprintf("[http] response: %s", formattedBody))
	}

	if err = json.Unmarshal(respData, ptr); err != nil {
		if globals.DebugMode {
			formattedBody := formatBodyForLog(respData, contentType)
			globals.Debug(fmt.Sprintf("[http] failed to decode response: %s\nresponse: %s", err, formattedBody))
		}

		return err
	}

	return nil
}

func HttpRaw(uri string, method string, headers map[string]string, body io.Reader, config []globals.ProxyConfig) (data []byte, err error) {
	if globals.DebugMode {
		formattedBody := ""
		if body != nil {
			if content, readErr := io.ReadAll(body); readErr == nil {
				formattedBody = formatBodyForLog(content, "")
				body = bytes.NewReader(content)
			} else {
				formattedBody = fmt.Sprintf("[Body Read Error] %s", readErr)
			}
		}
		globals.Debug(fmt.Sprintf("[http] %s %s\nheaders: \n%s\nbody: \n%s", method, uri, Marshal(headers), formattedBody))
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http] failed to create request: %s", err))
		}

		return nil, err
	}
	fillHeaders(req, headers)

	client := newClient(config)
	resp, err := client.Do(req)
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http] failed to send request: %s", err))
		}

		return nil, err
	}

	defer resp.Body.Close()

	if data, err = io.ReadAll(resp.Body); err != nil {
		if globals.DebugMode {
			contentType := resp.Header.Get("Content-Type")
			formattedBody := formatBodyForLog(data, contentType)
			globals.Debug(fmt.Sprintf("[http] failed to read response: %s\nresponse: %s", err, formattedBody))
		}

		return nil, err
	}

	if globals.DebugMode {
		contentType := resp.Header.Get("Content-Type")
		formattedBody := formatBodyForLog(data, contentType)
		globals.Debug(fmt.Sprintf("[http] response: %s", formattedBody))
	}
	return data, nil
}

func Get(uri string, headers map[string]string, config ...globals.ProxyConfig) (data interface{}, err error) {
	err = Http(uri, http.MethodGet, &data, headers, nil, config)
	return data, err
}

func GetRaw(uri string, headers map[string]string, config ...globals.ProxyConfig) (data string, err error) {
	buffer, err := HttpRaw(uri, http.MethodGet, headers, nil, config)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

func Post(uri string, headers map[string]string, body interface{}, config ...globals.ProxyConfig) (data interface{}, err error) {
	err = Http(uri, http.MethodPost, &data, headers, ConvertBody(body), config)
	return data, err
}

func ToString(data interface{}) string {
	switch v := data.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		data := Marshal(data)
		if len(data) > 0 {
			return data
		}

		return fmt.Sprintf("%v", data)
	}
}

func PostRaw(uri string, headers map[string]string, body interface{}, config ...globals.ProxyConfig) (data string, err error) {
	buffer, err := HttpRaw(uri, http.MethodPost, headers, ConvertBody(body), config)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

func ConvertBody(body interface{}) (form io.Reader) {
	if buffer, err := json.Marshal(body); err == nil {
		form = bytes.NewBuffer(buffer)
	}
	return form
}

func EventSource(method string, uri string, headers map[string]string, body interface{}, callback func(string) error, config ...globals.ProxyConfig) error {
	// panic recovery
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			globals.Warn(fmt.Sprintf("event source panic: %s (uri: %s, method: %s)\n%s", err, uri, method, stack))
		}
	}()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if globals.DebugMode {
		globals.Debug(fmt.Sprintf("[http-stream] %s %s\nheaders: \n%s\nbody: \n%s", method, uri, Marshal(headers), Marshal(body)))
	}

	client := newClient(config)
	req, err := http.NewRequest(method, uri, ConvertBody(body))
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http-stream] failed to create request: %s", err))
		}

		return err
	}

	fillHeaders(req, headers)

	res, err := client.Do(req)
	if err != nil {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http-stream] failed to send request: %s", err))
		}

		return err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		if globals.DebugMode {
			globals.Debug(fmt.Sprintf("[http-stream] request failed with status: %s\nresponse: %s", res.Status, res.Body))
		}

		if content, truncated, err := readErrorBody(res.Body); err == nil {
			if form, err := Unmarshal[map[string]interface{}](content); err == nil {
				data := MarshalWithIndent(form, 2)
				return fmt.Errorf("request failed with status: %s\n```json\n%s\n```", res.Status, appendTruncatedNotice(data, truncated, maxHTTPErrorBodyBytes))
			}

			return fmt.Errorf("request failed with status: %s\n%s", res.Status, appendTruncatedNotice(string(content), truncated, maxHTTPErrorBodyBytes))
		}

		return fmt.Errorf("request failed with status: %s", res.Status)
	}

	for {
		buf := make([]byte, 20480)
		n, err := res.Body.Read(buf)

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		data := string(buf[:n])
		for _, item := range strings.Split(data, "\n") {
			if globals.DebugMode {
				globals.Debug(fmt.Sprintf("[http-stream] response: %s", item))
			}

			segment := strings.TrimSpace(item)
			if len(segment) > 0 {
				if err := callback(segment); err != nil {
					return err
				}
			}
		}
	}
}
