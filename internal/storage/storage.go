package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nasymonk/img2svg/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

// 初始化业务数据库
func New(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "img2svg.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS convert_tasks (
			id TEXT PRIMARY KEY,
			original_name TEXT NOT NULL,
			input_path TEXT NOT NULL,
			output_path TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			params TEXT DEFAULT '{}',
			error_message TEXT DEFAULT '',
			created_at DATETIME DEFAULT (datetime('now')),
			finished_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_tasks_created ON convert_tasks(created_at DESC);
	`)
	return err
}

func (s *Store) CreateTask(t *models.ConvertTask) error {
	_, err := s.db.Exec(
		`INSERT INTO convert_tasks (id, original_name, input_path, output_path, status, progress, params, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		t.ID, t.OriginalName, t.InputPath, t.OutputPath, t.Status, t.Progress, t.Params,
	)
	return err
}

func (s *Store) UpdateTaskStatus(id, status string, progress int, outputPath, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE convert_tasks SET status=?, progress=?, output_path=?, error_message=?,
		 finished_at=CASE WHEN ? IN ('succeeded','failed') THEN datetime('now') ELSE finished_at END
		 WHERE id=?`,
		status, progress, outputPath, errMsg, status, id,
	)
	return err
}

func (s *Store) GetTask(id string) (*models.ConvertTask, error) {
	t := &models.ConvertTask{}
	var finished sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, original_name, input_path, output_path, status, progress, params, error_message, created_at, finished_at
		 FROM convert_tasks WHERE id=?`, id,
	).Scan(&t.ID, &t.OriginalName, &t.InputPath, &t.OutputPath,
		&t.Status, &t.Progress, &t.Params, &t.ErrorMessage, &t.CreatedAt, &finished)
	if err != nil {
		return nil, err
	}
	if finished.Valid {
		t.FinishedAt = &finished.Time
	}
	return t, nil
}

func (s *Store) ListTasks(limit int) ([]models.ConvertTask, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, original_name, input_path, output_path, status, progress, params, error_message, created_at, finished_at
		 FROM convert_tasks ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.ConvertTask
	for rows.Next() {
		t := models.ConvertTask{}
		var finished sql.NullTime
		if err := rows.Scan(&t.ID, &t.OriginalName, &t.InputPath, &t.OutputPath,
			&t.Status, &t.Progress, &t.Params, &t.ErrorMessage, &t.CreatedAt, &finished); err != nil {
			return nil, err
		}
		if finished.Valid {
			t.FinishedAt = &finished.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
