package store

import (
	"database/sql"
	"strings"
	"time"
)

type Material struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Style      string    `json:"style"`
	WorkflowID string    `json:"workflow_id"`
	Prompt     string    `json:"prompt"`
	Negative   string    `json:"negative,omitempty"`
	Seed       int64     `json:"seed"`
	Resolution int       `json:"resolution"`
	Favorite   bool      `json:"favorite"`
	Tags       []string  `json:"tags,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) IndexMaterial(m *Material) error {
	tags := strings.Join(m.Tags, " ")
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO materials (id, name, style, workflow_id, prompt, negative, seed, resolution, favorite, tags, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, style=excluded.style, prompt=excluded.prompt,
		   negative=excluded.negative, tags=excluded.tags`,
		m.ID, m.Name, m.Style, m.WorkflowID, m.Prompt, m.Negative,
		m.Seed, m.Resolution, boolInt(m.Favorite), tags, m.CreatedAt.UnixMilli()); err != nil {
		return err
	}
	if s.fts {
		if _, err := tx.Exec(`DELETE FROM materials_fts WHERE id=?`, m.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO materials_fts (id, name, prompt, tags) VALUES (?,?,?,?)`,
			m.ID, m.Name, m.Prompt, tags); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SetFavorite(id string, fav bool) error {
	_, err := s.db.Exec(`UPDATE materials SET favorite=? WHERE id=?`, boolInt(fav), id)
	return err
}

func (s *Store) DeleteMaterial(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM materials WHERE id=?`, id); err != nil {
		return err
	}
	if s.fts {
		if _, err := tx.Exec(`DELETE FROM materials_fts WHERE id=?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type MaterialQuery struct {
	Text          string
	Style         string
	FavoriteOnly  bool
	Limit, Offset int
}

func (s *Store) SearchMaterials(q MaterialQuery) ([]*Material, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 60
	}
	var (
		where []string
		args  []any
	)
	if t := strings.TrimSpace(q.Text); t != "" {
		if s.fts {
			// 前缀匹配更贴合"边打字边搜"的用法。
			where = append(where, `m.id IN (SELECT id FROM materials_fts WHERE materials_fts MATCH ?)`)
			args = append(args, ftsQuery(t))
		} else {
			where = append(where, `(m.name LIKE ? OR m.prompt LIKE ? OR m.tags LIKE ?)`)
			like := "%" + t + "%"
			args = append(args, like, like, like)
		}
	}
	if q.Style != "" {
		where = append(where, `m.style = ?`)
		args = append(args, q.Style)
	}
	if q.FavoriteOnly {
		where = append(where, `m.favorite = 1`)
	}

	sqlStr := `SELECT m.id, m.name, m.style, m.workflow_id, m.prompt, m.negative,
	                  m.seed, m.resolution, m.favorite, m.tags, m.created_at
	           FROM materials m`
	if len(where) > 0 {
		sqlStr += ` WHERE ` + strings.Join(where, " AND ")
	}
	sqlStr += ` ORDER BY m.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Material{}
	for rows.Next() {
		var (
			m       Material
			fav     int
			tags    string
			created int64
		)
		if err := rows.Scan(&m.ID, &m.Name, &m.Style, &m.WorkflowID, &m.Prompt, &m.Negative,
			&m.Seed, &m.Resolution, &fav, &tags, &created); err != nil {
			return nil, err
		}
		m.Favorite = fav != 0
		m.CreatedAt = time.UnixMilli(created)
		if tags != "" {
			m.Tags = strings.Fields(tags)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// GetMaterial 返回单条素材索引；不存在时返回 (nil, nil)。
func (s *Store) GetMaterial(id string) (*Material, error) {
	row := s.db.QueryRow(
		`SELECT id, name, style, workflow_id, prompt, negative, seed, resolution, favorite, tags, created_at
		 FROM materials WHERE id=?`, id)
	var (
		m       Material
		fav     int
		tags    string
		created int64
	)
	err := row.Scan(&m.ID, &m.Name, &m.Style, &m.WorkflowID, &m.Prompt, &m.Negative,
		&m.Seed, &m.Resolution, &fav, &tags, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Favorite = fav != 0
	m.CreatedAt = time.UnixMilli(created)
	if tags != "" {
		m.Tags = strings.Fields(tags)
	}
	return &m, nil
}

// ftsQuery 把用户输入转成 FTS5 表达式：逐词加前缀通配，并转义引号。
func ftsQuery(text string) string {
	fields := strings.Fields(text)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, `"`, `""`)
		parts = append(parts, `"`+f+`"*`)
	}
	return strings.Join(parts, " ")
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
