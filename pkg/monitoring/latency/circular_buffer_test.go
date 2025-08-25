package latency

import (
	"testing"
	"time"
)

func TestCircularBuffer_Add(t *testing.T) {
	// Test basic CircularBuffer functionality
	buffer := NewCircularBuffer(3)
	
	// Create test traces
	trace1 := &IntentTrace{
		ID:           "trace-1",
		IntentID:     "intent-1",
		IntentType:   "scaling",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(100 * time.Millisecond),
		TotalLatency: 100 * time.Millisecond,
	}
	
	trace2 := &IntentTrace{
		ID:           "trace-2",
		IntentID:     "intent-2",
		IntentType:   "scaling",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(200 * time.Millisecond),
		TotalLatency: 200 * time.Millisecond,
	}
	
	// Add traces to buffer
	buffer.Add(trace1)
	buffer.Add(trace2)
	
	// Verify buffer state
	if buffer.size != 2 {
		t.Errorf("Expected buffer size 2, got %d", buffer.size)
	}
	
	if buffer.capacity != 3 {
		t.Errorf("Expected buffer capacity 3, got %d", buffer.capacity)
	}
	
	// Verify traces are stored correctly
	if buffer.items[0] != trace1 {
		t.Errorf("Expected first item to be trace1")
	}
	
	if buffer.items[1] != trace2 {
		t.Errorf("Expected second item to be trace2")
	}
}

func TestCircularBuffer_OverflowBehavior(t *testing.T) {
	buffer := NewCircularBuffer(2)
	
	trace1 := &IntentTrace{ID: "trace-1"}
	trace2 := &IntentTrace{ID: "trace-2"}
	trace3 := &IntentTrace{ID: "trace-3"}
	
	// Fill buffer to capacity
	buffer.Add(trace1)
	buffer.Add(trace2)
	
	// Add one more item to test overflow behavior
	buffer.Add(trace3)
	
	// Buffer should still have size = capacity
	if buffer.size != 2 {
		t.Errorf("Expected buffer size 2 after overflow, got %d", buffer.size)
	}
	
	// The oldest item (trace1) should be replaced
	if buffer.items[0] != trace3 {
		t.Errorf("Expected first item to be trace3 after overflow")
	}
	
	if buffer.items[1] != trace2 {
		t.Errorf("Expected second item to be trace2 after overflow")
	}
}