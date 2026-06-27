package tui

import (
	"testing"

	"trainstation/agent"
	"trainstation/config"
	"trainstation/router"
)

func TestModelAppendAgentOutputNoPanic(t *testing.T) {
	cfg := config.Default()
	reg, _ := agent.NewRegistry(cfg)
	rtr, _ := router.New(cfg, reg)
	
	m := New(cfg, rtr, reg)
	
	// Add an agent entry
	m.addAgentEntry(1, "claude", "Test task")
	
	// Append output
	m.appendAgentOutput(1, "Hello ")
	
	// Simulate BubbleTea copying the model by value during Update
	mCopy := m
	
	// Append more output to the copy
	mCopy.appendAgentOutput(1, "World")
	
	// Verify it worked without panic
	if len(mCopy.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(mCopy.entries))
	}
	
	if mCopy.entries[1].text != "Hello World" {
		t.Errorf("expected text 'Hello World', got '%s'", mCopy.entries[1].text)
	}
}

func TestModelAddUserEntry(t *testing.T) {
	cfg := config.Default()
	reg, _ := agent.NewRegistry(cfg)
	rtr, _ := router.New(cfg, reg)
	
	m := New(cfg, rtr, reg)
	
	m.addUserEntry("This is a user test")
	
	if len(m.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.entries))
	}
	
	if m.entries[1].kind != entryUser {
		t.Errorf("expected kind %v, got %v", entryUser, m.entries[1].kind)
	}
	
	if m.entries[1].text != "This is a user test" {
		t.Errorf("expected text 'This is a user test', got '%s'", m.entries[1].text)
	}
}
