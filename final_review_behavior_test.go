package stoplore_test

import (
	"context"
	"encoding/csv"
	stoplore "github.com/saintmili/stoplore"
	"os"
	"path/filepath"
	"testing"
)

func TestJournalRejectsMissingPersistedCollections(t *testing.T) {
	for _, document := range []string{`{"version":1}`, `{"version":1,"exchanges":{},"accounts":{}}`, `{"version":1,"exchanges":null,"accounts":{},"records":{}}`} {
		t.Run(document, func(t *testing.T) {
			dir := t.TempDir()
			f, err := os.Create(filepath.Join(dir, "mtl_journal.csv"))
			if err != nil {
				t.Fatal(err)
			}
			w := csv.NewWriter(f)
			if err = w.Write([]string{"MTL_JOURNAL_V1", document}); err != nil {
				t.Fatal(err)
			}
			w.Flush()
			f.Close()
			s := stoplore.NewWithCSV(dir)
			if err = s.Init(); err != nil {
				return
			}
			if _, err = s.ListTrades(context.Background()); err == nil {
				t.Fatal("incomplete persisted journal was accepted")
			}
		})
	}
}
