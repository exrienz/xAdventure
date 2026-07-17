package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/muz/xadventure/internal/domain"
)

func TestSQLiteRepo_CRUD(t *testing.T) {
	dbPath := "./test_adventure.db"
	defer os.Remove(dbPath)

	repo, err := NewSQLiteRepo(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	// Run migrations for test
	migrationContent, err := os.ReadFile("../../migrations/001_init.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration: %v", err)
	}
	db := repo.(*SQLiteRepo).DB()
	if _, err := db.Exec(string(migrationContent)); err != nil {
		t.Fatalf("failed to run migration: %v", err)
	}
	
	migrationContent2, err := os.ReadFile("../../migrations/002_add_color_coded_content.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration 2: %v", err)
	}
	if _, err := db.Exec(string(migrationContent2)); err != nil {
		t.Fatalf("failed to run migration 2: %v", err)
	}

	ctx := context.Background()

	session := &domain.Session{
		ID:        uuid.New().String(),
		UserName:  "Test",
		Genre:     "Adventure",
		Age:       20,
		Gender:    "Male",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		State: domain.GameState{
			Health:    100,
			MaxHealth: 100,
			Inventory: []string{"Sword"},
			Stats:     map[string]int{"strength": 10},
		},
		CurrentChoices: []string{"A", "B", "C", "D"},
	}

	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	fetched, err := repo.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected session, got nil")
	}
	if fetched.UserName != "Test" {
		t.Fatalf("expected Test, got %s", fetched.UserName)
	}
	if len(fetched.CurrentChoices) != 4 {
		t.Fatalf("expected 4 choices, got %d", len(fetched.CurrentChoices))
	}

	fetched.State.Health = 80
	if err := repo.UpdateSession(ctx, fetched); err != nil {
		t.Fatalf("failed to update session: %v", err)
	}

	log := &domain.StoryLog{
		SessionID:  session.ID,
		TurnNumber: 1,
		Content:    "Test story",
		ChoiceMade: "A",
		Timestamp:  time.Now(),
	}
	if err := repo.AppendStoryLog(ctx, log); err != nil {
		t.Fatalf("failed to append log: %v", err)
	}

	logs, err := repo.GetStoryLogs(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
}
