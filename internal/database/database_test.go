package database

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

func tableExists(t *testing.T, db *DB, tableName string) bool {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err != nil {
		return false
	}
	return name == tableName
}

func TestNewDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	logger := zap.NewNop()

	db, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected db instance, got nil")
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}

	// Verify we can ping
	if err := db.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestTableInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_init.db")
	logger := zap.NewNop()

	db, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	coreTables := []string{
		"conversations",
		"messages",
		"process_details",
		"tool_executions",
		"tool_stats",
		"skill_stats",
		"attack_chain_nodes",
		"attack_chain_edges",
	}

	for _, table := range coreTables {
		if !tableExists(t, db, table) {
			t.Errorf("Expected table %s to exist, but it doesn't", table)
		}
	}
}

func TestTableInitialization_Extra(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_init_extra.db")
	logger := zap.NewNop()

	db, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	extraTables := []string{
		"knowledge_retrieval_logs",
		"conversation_groups",
		"conversation_group_mappings",
		"vulnerabilities",
		"batch_task_queues",
		"batch_tasks",
		"webshell_connections",
		"webshell_connection_states",
	}

	for _, table := range extraTables {
		if !tableExists(t, db, table) {
			t.Errorf("Expected table %s to exist, but it doesn't", table)
		}
	}
}

func TestNewKnowledgeDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_knowledge.db")
	logger := zap.NewNop()

	db, err := NewKnowledgeDB(dbPath, logger)
	if err != nil {
		t.Fatalf("NewKnowledgeDB failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected db instance, got nil")
	}

	knowledgeTables := []string{
		"knowledge_base_items",
		"knowledge_embeddings",
		"knowledge_retrieval_logs",
	}

	for _, table := range knowledgeTables {
		if !tableExists(t, db, table) {
			t.Errorf("Expected table %s to exist in knowledge DB, but it doesn't", table)
		}
	}
}

func TestDB_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_close.db")
	logger := zap.NewNop()

	db, err := NewDB(dbPath, logger)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Ping should fail after close
	if err := db.Ping(); err == nil {
		t.Error("Ping succeeded after Close, expected error")
	}
}
