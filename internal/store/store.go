// Package store 提供 SQLite 持久化：任务状态与素材索引。
//
// 用 modernc.org/sqlite（纯 Go 实现），免去 cgo 与 Windows 上的编译器依赖，
// 保证"一个 exe 就能跑"。
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	// fts 记录 FTS5 是否可用；不可用时素材搜索退化为 LIKE。
	fts bool
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	// SQLite 单写者；并发写会互相阻塞，限制连接数比让它们排队报错更可控。
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// HasFTS 表示素材搜索是否走 FTS5 索引。
func (s *Store) HasFTS() bool { return s.fts }

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS jobs (
  id           TEXT PRIMARY KEY,
  material_id  TEXT NOT NULL,
  workflow_id  TEXT NOT NULL,
  params       TEXT NOT NULL,
  status       TEXT NOT NULL,
  prompt_id    TEXT NOT NULL DEFAULT '',
  progress     REAL NOT NULL DEFAULT 0,
  stage        TEXT NOT NULL DEFAULT '',
  error        TEXT NOT NULL DEFAULT '',
  batch_id     TEXT NOT NULL DEFAULT '',
  created_at   INTEGER NOT NULL,
  started_at   INTEGER,
  finished_at  INTEGER
);
CREATE INDEX IF NOT EXISTS idx_jobs_status  ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_batch   ON jobs(batch_id);

CREATE TABLE IF NOT EXISTS materials (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  style       TEXT NOT NULL,
  workflow_id TEXT NOT NULL,
  prompt      TEXT NOT NULL,
  negative    TEXT NOT NULL DEFAULT '',
  seed        INTEGER NOT NULL DEFAULT 0,
  resolution  INTEGER NOT NULL DEFAULT 0,
  favorite    INTEGER NOT NULL DEFAULT 0,
  tags        TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_materials_created  ON materials(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_materials_favorite ON materials(favorite);
CREATE INDEX IF NOT EXISTS idx_materials_style    ON materials(style);

-- 图片：单张出图，不是材质套装。
--
-- 单开一张表而不是塞进 materials：材质套装有 manifest、多路通道、无缝与
-- 法线方向这些契约，UE 那边照着读；一张只有底色、tileable=false 的"材质"
-- 是在骗下游。两者的检索维度也不一样（图片关心尺寸与花费，材质关心风格
-- 与分辨率）。
CREATE TABLE IF NOT EXISTS pictures (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  workflow_id TEXT NOT NULL,
  prompt      TEXT NOT NULL,
  negative    TEXT NOT NULL DEFAULT '',
  seed        INTEGER NOT NULL DEFAULT 0,
  width       INTEGER NOT NULL DEFAULT 0,
  height      INTEGER NOT NULL DEFAULT 0,
  provider    TEXT NOT NULL DEFAULT '',
  model       TEXT NOT NULL DEFAULT '',
  cost_usd    REAL NOT NULL DEFAULT 0,
  favorite    INTEGER NOT NULL DEFAULT 0,
  tags        TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pictures_created  ON pictures(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pictures_favorite ON pictures(favorite);

-- 参考图库：上传的与从产物提升上来的参考图。
--
-- 之前参考图是"传一张、用完就散"，没有留存。做成库之后可以反复用同一张，
-- 也能把满意的产物直接变成下一轮的参考。
CREATE TABLE IF NOT EXISTS refs (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  file        TEXT NOT NULL,
  -- comfy_name 是它在 ComfyUI input 目录里的文件名。
  -- 单独记是因为提交工作流时要填的是那个名字，而不是我们自己的路径。
  comfy_name  TEXT NOT NULL DEFAULT '',
  width       INTEGER NOT NULL DEFAULT 0,
  height      INTEGER NOT NULL DEFAULT 0,
  bytes       INTEGER NOT NULL DEFAULT 0,
  origin      TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refs_created ON refs(created_at DESC);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("建表: %w", err)
	}

	// FTS5 未必编译进当前构建；探测一次，失败就降级而不是让整个程序起不来。
	const ftsSchema = `
CREATE VIRTUAL TABLE IF NOT EXISTS materials_fts USING fts5(
  id UNINDEXED, name, prompt, tags, tokenize='unicode61'
);`
	if _, err := s.db.Exec(ftsSchema); err == nil {
		s.fts = true
	}
	return nil
}

// ---------- 任务 ----------

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

func (s Status) Terminal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCanceled
}

type Job struct {
	ID         string         `json:"id"`
	MaterialID string         `json:"material_id"`
	WorkflowID string         `json:"workflow_id"`
	BatchID    string         `json:"batch_id,omitempty"`
	Params     map[string]any `json:"params"`
	Status     Status         `json:"status"`
	PromptID   string         `json:"prompt_id,omitempty"`
	Progress   float64        `json:"progress"`
	Stage      string         `json:"stage,omitempty"`
	Error      string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
}

func (s *Store) CreateJob(j *Job) error {
	p, err := json.Marshal(j.Params)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO jobs (id, material_id, workflow_id, params, status, batch_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.MaterialID, j.WorkflowID, string(p), string(j.Status), j.BatchID, j.CreatedAt.UnixMilli())
	return err
}

func (s *Store) UpdateJob(j *Job) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status=?, prompt_id=?, progress=?, stage=?, error=?, started_at=?, finished_at=?
		 WHERE id=?`,
		string(j.Status), j.PromptID, j.Progress, j.Stage, j.Error,
		msPtr(j.StartedAt), msPtr(j.FinishedAt), j.ID)
	return err
}

func (s *Store) GetJob(id string) (*Job, error) {
	row := s.db.QueryRow(`SELECT `+jobCols+` FROM jobs WHERE id=?`, id)
	return scanJob(row)
}

// ListJobs 返回最近的任务，status 为空表示不过滤。
func (s *Store) ListJobs(status Status, limit int) ([]*Job, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT ` + jobCols + ` FROM jobs`
	args := []any{}
	if status != "" {
		q += ` WHERE status=?`
		args = append(args, string(status))
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ResumeInterrupted 把上次进程退出时残留的 queued/running 任务标记为失败。
//
// 单 GPU 单 worker 的前提下，进程重启后原来的 ComfyUI 任务已经无从跟踪，
// 与其让它们永远挂着，不如明确判失败并说明原因，用户可以一键重来。
func (s *Store) ResumeInterrupted(reason string) (int, error) {
	now := time.Now().UnixMilli()
	res, err := s.db.Exec(
		`UPDATE jobs SET status=?, error=?, finished_at=? WHERE status IN (?, ?)`,
		string(StatusFailed), reason, now, string(StatusQueued), string(StatusRunning))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

const jobCols = `id, material_id, workflow_id, params, status, prompt_id, progress, stage, error, batch_id, created_at, started_at, finished_at`

type scanner interface{ Scan(...any) error }

func scanJob(r scanner) (*Job, error) {
	var (
		j                 Job
		params, status    string
		created           int64
		started, finished sql.NullInt64
	)
	err := r.Scan(&j.ID, &j.MaterialID, &j.WorkflowID, &params, &status, &j.PromptID,
		&j.Progress, &j.Stage, &j.Error, &j.BatchID, &created, &started, &finished)
	if err != nil {
		return nil, err
	}
	j.Status = Status(status)
	j.CreatedAt = time.UnixMilli(created)
	if started.Valid {
		t := time.UnixMilli(started.Int64)
		j.StartedAt = &t
	}
	if finished.Valid {
		t := time.UnixMilli(finished.Int64)
		j.FinishedAt = &t
	}
	if params != "" {
		_ = json.Unmarshal([]byte(params), &j.Params)
	}
	return &j, nil
}

func msPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UnixMilli()
}
