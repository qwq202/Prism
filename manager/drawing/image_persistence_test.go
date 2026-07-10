package drawing

import (
	"errors"
	"testing"
)

func TestGeneratedImagesFromContentPersistsRemoteSources(t *testing.T) {
	previous := persistRemoteDrawingImageSource
	t.Cleanup(func() {
		persistRemoteDrawingImageSource = previous
	})

	const source = "https://images.example.com/provider-result.png"
	const stored = "/attachments/0123456789abcdef0123456789abcdef.png"
	called := false
	persistRemoteDrawingImageSource = func(value string) (string, error) {
		called = true
		if value != source {
			t.Fatalf("expected source %q, got %q", source, value)
		}
		return stored, nil
	}

	images, err := generatedImagesFromContent("![image]("+source+")", "draw a cat", "task-1")
	if err != nil {
		t.Fatalf("generate images from content: %v", err)
	}
	if !called {
		t.Fatalf("expected remote image source to be persisted")
	}
	if len(images) != 1 || images[0].Src != stored {
		t.Fatalf("expected durable generated image, got %#v", images)
	}
}

func TestGeneratedImagesFromContentKeepsOwnedAttachments(t *testing.T) {
	previous := persistRemoteDrawingImageSource
	t.Cleanup(func() {
		persistRemoteDrawingImageSource = previous
	})
	persistRemoteDrawingImageSource = func(string) (string, error) {
		return "", errors.New("owned attachment should not be persisted again")
	}

	const stored = "/attachments/0123456789abcdef0123456789abcdef.png"
	images, err := generatedImagesFromContent("![image]("+stored+")", "draw a cat", "task-1")
	if err != nil {
		t.Fatalf("generate images from owned attachment: %v", err)
	}
	if len(images) != 1 || images[0].Src != stored {
		t.Fatalf("expected owned attachment to remain unchanged, got %#v", images)
	}
}

func TestGeneratedImagesFromContentReturnsPersistenceError(t *testing.T) {
	previous := persistRemoteDrawingImageSource
	t.Cleanup(func() {
		persistRemoteDrawingImageSource = previous
	})

	expected := errors.New("remote image expired")
	persistRemoteDrawingImageSource = func(string) (string, error) {
		return "", expected
	}

	images, err := generatedImagesFromContent(
		"![image](https://images.example.com/provider-result.png)",
		"draw a cat",
		"task-1",
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected persistence error, got images=%#v err=%v", images, err)
	}
}
