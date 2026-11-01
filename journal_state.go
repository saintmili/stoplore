package mtl

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type journalState struct {
	Version   int                        `json:"version"`
	Exchanges map[string]Exchange        `json:"exchanges"`
	Accounts  map[string]Account         `json:"accounts"`
	Records   map[string]json.RawMessage `json:"records"`
}

func emptyJournalState() *journalState {
	return &journalState{Version: 1, Exchanges: map[string]Exchange{}, Accounts: map[string]Account{}, Records: map[string]json.RawMessage{}}
}
func (s *journalState) get(key string, dest any) error {
	v, ok := s.Records[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(v, dest)
}
func (s *journalState) put(key string, value any) error {
	v, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.Records[key] = v
	return nil
}
func decodeState(data []byte) (*journalState, error) {
	s := &journalState{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("journal state: %w", err)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("unsupported journal version %d", s.Version)
	}
	if s.Exchanges == nil || s.Accounts == nil || s.Records == nil {
		return nil, fmt.Errorf("journal state is missing required collections")
	}
	return s, nil
}

type documentStore interface {
	init(context.Context) error
	transact(context.Context, func(*journalState) error, bool) (*journalState, error)
}

func (s *Service) read(ctx context.Context) (*journalState, error) {
	return s.documents.transact(ctx, nil, false)
}
func (s *Service) update(ctx context.Context, change func(*journalState) error) error {
	_, err := s.documents.transact(ctx, change, true)
	return err
}

type csvDocuments struct{ dir string }

func (c *csvDocuments) init(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.MkdirAll(c.dir, 0755)
}
func (c *csvDocuments) transact(ctx context.Context, change func(*journalState) error, write bool) (*journalState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(c.dir, ".mtl-journal.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	state := emptyJournalState()
	path := filepath.Join(c.dir, "mtl_journal.csv")
	f, err := os.Open(path)
	if err == nil {
		r := csv.NewReader(f)
		record, e := r.Read()
		if e != nil {
			f.Close()
			return nil, fmt.Errorf("read journal: %w", e)
		}
		if len(record) != 2 || record[0] != "MTL_JOURNAL_V1" {
			f.Close()
			return nil, fmt.Errorf("invalid journal CSV document")
		}
		if _, e = r.Read(); e != io.EOF {
			f.Close()
			return nil, fmt.Errorf("journal CSV must contain one document")
		}
		f.Close()
		state, err = decodeState([]byte(record[1]))
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if change != nil {
		if err = change(state); err != nil {
			return nil, err
		}
	}
	if !write {
		return state, nil
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(c.dir, ".mtl-journal-*.csv")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	w := csv.NewWriter(tmp)
	if err = w.Write([]string{"MTL_JOURNAL_V1", string(data)}); err != nil {
		return nil, err
	}
	w.Flush()
	if err = w.Error(); err != nil {
		return nil, err
	}
	if err = tmp.Sync(); err != nil {
		return nil, err
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return nil, err
	}
	return state, nil
}

type postgresDocuments struct{ db *pgxpool.Pool }

func (p *postgresDocuments) init(ctx context.Context) error {
	_, err := p.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS mtl_journal_state (id INTEGER PRIMARY KEY CHECK (id=1), document JSONB NOT NULL); INSERT INTO mtl_journal_state(id,document) VALUES(1,'{"version":1,"exchanges":{},"accounts":{},"records":{}}') ON CONFLICT DO NOTHING`)
	return err
}
func (p *postgresDocuments) transact(ctx context.Context, change func(*journalState) error, write bool) (*journalState, error) {
	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	query := "SELECT document FROM mtl_journal_state WHERE id=1"
	if write {
		query += " FOR UPDATE"
	}
	var data []byte
	if err = tx.QueryRow(ctx, query).Scan(&data); err != nil {
		return nil, err
	}
	state, err := decodeState(data)
	if err != nil {
		return nil, err
	}
	if change != nil {
		if err = change(state); err != nil {
			return nil, err
		}
	}
	if write {
		data, err = json.Marshal(state)
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, "UPDATE mtl_journal_state SET document=$1 WHERE id=1", data); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return state, nil
}
