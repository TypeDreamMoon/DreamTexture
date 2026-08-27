package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Ref 是参考图库里的一条。
//
// 之前参考图是"传一张、用完就散"，没有留存——同一张图要反复用就得反复传，
// 而且满意的产物没法直接变成下一轮的参考。
type Ref struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// File 是我们自己库里的文件名。
	File string `json:"file"`
	// ComfyName 是它在 ComfyUI input 目录里的文件名。
	//
	// 单独记是因为提交工作流时 LoadImage 要填的是那个名字，不是我们的路径；
	// 而 ComfyUI 可能被清空或换了实例，那时这个字段会失效、需要重新上传。
	ComfyName string    `json:"comfy_name,omitempty"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Bytes     int64     `json:"bytes"`
	Origin    string    `json:"origin,omitempty"` // upload | picture:<id> | material:<id>
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) AddRef(r *Ref) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO refs (id, name, file, comfy_name, width, height, bytes, origin, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ID, r.Name, r.File, r.ComfyName, r.Width, r.Height, r.Bytes, r.Origin,
		r.CreatedAt.UnixMilli())
	return err
}

func (s *Store) ListRefs(limit int) ([]*Ref, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT id, name, file, comfy_name, width, height, bytes, origin, created_at
		 FROM refs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Ref{}
	for rows.Next() {
		r, err := scanRef(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetRef(id string) (*Ref, error) {
	row := s.db.QueryRow(
		`SELECT id, name, file, comfy_name, width, height, bytes, origin, created_at
		 FROM refs WHERE id=?`, id)
	r, err := scanRef(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (s *Store) RenameRef(id, name string) error {
	res, err := s.db.Exec(`UPDATE refs SET name=? WHERE id=?`, name, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("参考图 %s 不存在", id)
	}
	return nil
}

// SetRefComfyName 记下它在 ComfyUI 里的文件名。
// ComfyUI 被清空或换实例后要重新上传，这个字段随之更新。
func (s *Store) SetRefComfyName(id, name string) error {
	_, err := s.db.Exec(`UPDATE refs SET comfy_name=? WHERE id=?`, name, id)
	return err
}

func (s *Store) DeleteRef(id string) error {
	_, err := s.db.Exec(`DELETE FROM refs WHERE id=?`, id)
	return err
}

func scanRef(r scanner) (*Ref, error) {
	var (
		x  Ref
		ms int64
	)
	if err := r.Scan(&x.ID, &x.Name, &x.File, &x.ComfyName, &x.Width, &x.Height,
		&x.Bytes, &x.Origin, &ms); err != nil {
		return nil, err
	}
	x.CreatedAt = time.UnixMilli(ms)
	return &x, nil
}
