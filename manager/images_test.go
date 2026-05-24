package manager

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func attachmentUploadResponse(t *testing.T, filename, contentType string, content []byte, contentLength int64) AttachmentUploadResponse {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/attachment/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Set("user", "tester")

	UploadAttachmentAPI(ctx)

	var response AttachmentUploadResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}

func TestUploadAttachmentRejectsOversizedBody(t *testing.T) {
	response := attachmentUploadResponse(
		t,
		"tiny.png",
		"image/png",
		[]byte("\x89PNG\r\n\x1a\n"),
		maxAttachmentMultipartUploadBytes+1,
	)

	if response.Status {
		t.Fatalf("expected oversized upload to fail")
	}
	if !strings.Contains(response.Error, "100MB") {
		t.Fatalf("expected size-limit error, got %q", response.Error)
	}
}

func TestUploadAttachmentRejectsSVG(t *testing.T) {
	response := attachmentUploadResponse(
		t,
		"unsafe.svg",
		"image/svg+xml",
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		-1,
	)

	if response.Status {
		t.Fatalf("expected svg upload to fail")
	}
	if response.Error != "only raster image upload is supported" {
		t.Fatalf("unexpected error: %q", response.Error)
	}
}

func TestUploadAttachmentRejectsSpoofedImage(t *testing.T) {
	response := attachmentUploadResponse(
		t,
		"fake.png",
		"image/png",
		[]byte("this is not an image"),
		-1,
	)

	if response.Status {
		t.Fatalf("expected spoofed image upload to fail")
	}
	if response.Error != "invalid image data" {
		t.Fatalf("unexpected error: %q", response.Error)
	}
}
