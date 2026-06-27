package db

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Prompt struct {
	ID        uint64     `db:"id" json:"id"`
	Text      string     `db:"text" json:"text"`
	Category  string     `db:"category" json:"category"`
	Active    bool       `db:"active" json:"active"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	RetiredAt *time.Time `db:"retired_at" json:"retired_at,omitempty"`
	Notes     string     `db:"notes" json:"notes"`
}

type PromptRepo struct {
	db *sqlx.DB
}

func NewPromptRepo(db *sqlx.DB) *PromptRepo {
	return &PromptRepo{db: db}
}

func (r *PromptRepo) ListActive() ([]Prompt, error) {
	var prompts []Prompt
	err := r.db.Select(&prompts, "SELECT id, text, category, active, created_at, retired_at, notes FROM prompts WHERE active = TRUE")
	return prompts, err
}

func (r *PromptRepo) Create(p *Prompt) error {
	query := `INSERT INTO prompts (text, category, active, notes) VALUES (:text, :category, :active, :notes)`
	res, err := r.db.NamedExec(query, p)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		p.ID = uint64(id)
	}
	return err
}

func (r *PromptRepo) Retire(id uint64) error {
	_, err := r.db.Exec("UPDATE prompts SET active = FALSE, retired_at = NOW() WHERE id = ?", id)
	return err
}

func (r *PromptRepo) BulkInsert(prompts []Prompt) error {
	query := `INSERT INTO prompts (text, category, active, notes) VALUES (:text, :category, :active, :notes)`
	_, err := r.db.NamedExec(query, prompts)
	return err
}
