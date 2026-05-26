package manager

import (
	"chat/globals"
	"chat/manager/conversation"
	"testing"
	"time"
)

func TestLatestMessageContentHandlesEmptySegment(t *testing.T) {
	if content, ok := latestMessageContent(nil); ok || content != "" {
		t.Fatalf("expected empty segment to be rejected, got content=%q ok=%v", content, ok)
	}

	content, ok := latestMessageContent([]globals.Message{
		{Role: globals.User, Content: "first"},
		{Role: globals.User, Content: "latest"},
	})
	if !ok || content != "latest" {
		t.Fatalf("expected latest message content, got content=%q ok=%v", content, ok)
	}
}

func TestCreateStopSignalEmitsStopAndCancelsPolling(t *testing.T) {
	conn := NewConnection(nil, false, "", 2)
	conn.Write(&conversation.FormMessage{Type: StopType})

	stopSignal, cancel := createStopSignal(conn, nil)
	defer cancel()

	select {
	case stopped := <-stopSignal:
		if !stopped {
			t.Fatalf("expected stop signal")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for stop signal")
	}

	cancel()
}

func TestCreateStopSignalHandlesRemoveWithoutStopping(t *testing.T) {
	conn := NewConnection(nil, false, "", 3)
	conn.Write(&conversation.FormMessage{Type: RemoveType, Message: "2"})

	removed := make(chan int, 1)
	stopSignal, cancel := createStopSignal(conn, func(index int) {
		removed <- index
	})
	defer cancel()

	select {
	case index := <-removed:
		if index != 2 {
			t.Fatalf("expected remove index 2, got %d", index)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for remove handler")
	}

	select {
	case stopped := <-stopSignal:
		if stopped {
			t.Fatalf("remove event should not stop chat request")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for non-stop signal")
	}
}

func TestCreateStopSignalConsumesStopAfterRemove(t *testing.T) {
	conn := NewConnection(nil, false, "", 3)
	conn.Write(&conversation.FormMessage{Type: RemoveType, Message: "1"})
	conn.Write(&conversation.FormMessage{Type: StopType})

	removed := make(chan int, 1)
	stopSignal, cancel := createStopSignal(conn, func(index int) {
		removed <- index
	})
	defer cancel()

	select {
	case index := <-removed:
		if index != 1 {
			t.Fatalf("expected remove index 1, got %d", index)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for remove handler")
	}

	select {
	case stopped := <-stopSignal:
		if !stopped {
			t.Fatalf("expected stop signal after queued remove event")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for stop signal")
	}
}
