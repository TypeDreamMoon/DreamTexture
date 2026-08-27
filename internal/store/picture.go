package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Picture 是一张生成出来的图片的索引记录。
//
// 与 Material 并列而不是合并：材质套装有 manifest、多路通道、无缝与法线方向
// 这些给 UE 的契约，图片没有；反过来图片关心尺寸与花费，材质不关心。
type Picture struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	WorkflowID string    `json:"workflow_id"`
	Prompt     string    `json:"prompt"`
	Negative   string    `json:"negative,omitempty"`
	Seed       int64     `json:"seed"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Provider   string    `json:"provider,omitempty"`
	Model      string    `json:"model,omitempty"`
	CostUSD    float64   `json:"cost_usd,omitempty"`
	Favorite   bool      `json:"favorite"`
	Tags       []string  `json:"tags,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) IndexPicture(p *Picture) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO pictures
		 (id, name, workflow_id, prompt, negative, seed, width, height,
		  provider, model, cost_usd, favorite, tags, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,
		         COALESCE((SELECT favorite FROM pictures WHERE id=?), 0), ?, ?)`,
		p.ID, p.Name, p.WorkflowID, p.Prompt, p.Negative, p.Seed, p.Width, p.Height,
		p.Provider, p.Model, p.CostUSD, p.ID, strings.Join(p.Tags, ","),
		p.CreatedAt.UnixMilli())
	return err
}

type PictureQuery struct {
	Q     string
	Fav   bool
	Limit int
}

func (s *Store) SearchPictures(q PictureQuery) ([]*Picture, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 120
	}
	var (
		where []string
		args  []any
	)
	if t := strings.TrimSpace(q.Q); t != "" {
		// 图片量级远小于素材，直接 LIKE 就够，不必再上一套 FTS 表。
		where = append(where, "(name LIKE ? OR prompt LIKE ?)")
		args = append(args, "%"+t+"%", "%"+t+"%")
	}
	if q.Fav {
		where = append(where, "favorite = 1")
	}
	sqlText := `SELECT id, name, workflow_id, prompt, negative, seed, width, height,
	                   provider, model, cost_usd, favorite, tags, created_at FROM pictures`
	if len(where) > 0 {
		sqlText += " WHERE " + strings.Join(where, " AND ")
	}
	sqlText += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, q.Limit)

	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPictures(rows)
}

func (s *Store) GetPicture(id string) (*Picture, error) {
	row := s.db.QueryRow(
		`SELECT id, name, workflow_id, prompt, negative, seed, width, height,
		        provider, model, cost_usd, favorite, tags, created_at
		 FROM pictures WHERE id=?`, id)
	p, err := scanPicture(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *Store) SetPictureFavorite(id string, fav bool) error {
	res, err := s.db.Exec(`UPDATE pictures SET favorite=? WHERE id=?`, boolInt(fav), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("图片 %s 不存在", id)
	}
	return nil
}

func (s *Store) DeletePicture(id string) error {
	_, err := s.db.Exec(`DELETE FROM pictures WHERE id=?`, id)
	return err
}

func scanPicture(r scanner) (*Picture, error) {
	var (
		p    Picture
		fav  int
		tags string
		ms   int64
	)
	if err := r.Scan(&p.ID, &p.Name, &p.WorkflowID, &p.Prompt, &p.Negative, &p.Seed,
		&p.Width, &p.Height, &p.Provider, &p.Model, &p.CostUSD, &fav, &tags, &ms); err != nil {
		return nil, err
	}
	p.Favorite = fav != 0
	if tags != "" {
		p.Tags = strings.Split(tags, ",")
	}
	p.CreatedAt = time.UnixMilli(ms)
	return &p, nil
}

func scanPictures(rows *sql.Rows) ([]*Picture, error) {
	out := []*Picture{}
	for rows.Next() {
		p, err := scanPicture(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
