package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/muz/xadventure/internal/domain"
	_ "modernc.org/sqlite"
)

type Repository interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id string) (*domain.Session, error)
	UpdateSession(ctx context.Context, session *domain.Session) error
	AppendStoryLog(ctx context.Context, log *domain.StoryLog) error
	GetStoryLogs(ctx context.Context, sessionID string) ([]domain.StoryLog, error)
	Close() error
}

type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(dbPath string) (Repository, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SQLiteRepo{db: db}, nil
}

func (r *SQLiteRepo) CreateSession(ctx context.Context, session *domain.Session) error {
	stateJSON, err := json.Marshal(session.State)
	if err != nil {
		return err
	}
	choicesJSON, err := json.Marshal(session.CurrentChoices)
	if err != nil {
		return err
	}
	query := `INSERT INTO sessions (id, user_name, genre, age, gender, archetype, seed, state, current_choices, status, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query,
		session.ID, session.UserName, session.Genre, session.Age, session.Gender,
		session.Archetype, session.Seed, string(stateJSON), string(choicesJSON),
		session.Status, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

func (r *SQLiteRepo) GetSession(ctx context.Context, id string) (*domain.Session, error) {
	query := `SELECT id, user_name, genre, age, gender, archetype, seed, state, current_choices, status, created_at, updated_at FROM sessions WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var s domain.Session
	var stateStr, choicesStr string
	err := row.Scan(
		&s.ID, &s.UserName, &s.Genre, &s.Age, &s.Gender, &s.Archetype, &s.Seed,
		&stateStr, &choicesStr, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(stateStr), &s.State); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(choicesStr), &s.CurrentChoices); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteRepo) UpdateSession(ctx context.Context, session *domain.Session) error {
	stateJSON, err := json.Marshal(session.State)
	if err != nil {
		return err
	}
	choicesJSON, err := json.Marshal(session.CurrentChoices)
	if err != nil {
		return err
	}
	query := `UPDATE sessions SET state = ?, current_choices = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query, string(stateJSON), string(choicesJSON), session.Status, session.ID)
	return err
}

func (r *SQLiteRepo) AppendStoryLog(ctx context.Context, log *domain.StoryLog) error {
	query := `INSERT INTO story_logs (session_id, turn_number, content, color_coded_content, chapter_title, choice_made, timestamp) 
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, log.SessionID, log.TurnNumber, log.Content, log.ColorCodedContent, log.ChapterTitle, log.ChoiceMade, log.Timestamp)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	log.ID = int(id)
	return nil
}

func (r *SQLiteRepo) GetStoryLogs(ctx context.Context, sessionID string) ([]domain.StoryLog, error) {
	query := `SELECT id, session_id, turn_number, content, color_coded_content, chapter_title, choice_made, timestamp FROM story_logs WHERE session_id = ? ORDER BY turn_number ASC`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.StoryLog
	for rows.Next() {
		var l domain.StoryLog
		var choice, chapter, colorCoded sql.NullString
		if err := rows.Scan(&l.ID, &l.SessionID, &l.TurnNumber, &l.Content, &colorCoded, &chapter, &choice, &l.Timestamp); err != nil {
			return nil, err
		}
		l.ColorCodedContent = colorCoded.String
		l.ChoiceMade = choice.String
		l.ChapterTitle = chapter.String
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}

// DB exposes the underlying sql.DB for testing purposes.
func (r *SQLiteRepo) DB() *sql.DB {
	return r.db
}
