package mtl_test

import (
	"bytes"
	"context"
	"encoding/csv"
	mtl "github.com/saintmili/mtl"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var importHeader = []string{"ID", "EXCHANGE", "STRATEGY", "SYMBOL", "SIDE", "ORDERTYPE", "ENTRYTYPE", "LEVERAGE", "TIMEFRAME", "ENTRYPRICE", "ENTRYMENTALSTATE", "POSITIONMARGIN", "ENTRYFEERATE", "STOPLOSSPRICE", "TAKEPROFITPRICE", "ENTRYIMAGEURL", "OPENEDAT", "EXITPRICE", "EXITMARGINPERCENT", "EXITFEERATE", "EXITMENTALSTATE", "EXITIMAGEURL", "EXITEDAT", "NOTES", "CREATEDAT", "UPDATEDAT"}

func syntheticImportRow() []string {
	return []string{"c1c0c94b-41dc-4450-9be4-2e67c6078f11", "Old venue", "Breakout", "BTCUSDT", "BUY", "LIMIT", "TRADE", "2", "1h", "100", "calm", "50", "0.001", "90", "120", "entry.png", "2026-01-01T10:00:00.123456789+03:30", "120", "100", "0.002", "happy", "exit.png", "2026-01-02T10:00:00.123456789+03:30", "synthetic, \"notes\"\nsecond line", "2026-01-01T09:00:00Z", "2026-01-02T09:00:00Z"}
}
func importFixture(t *testing.T, header, row []string) (string, []byte) {
	t.Helper()
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.WriteAll([][]string{header, row}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "positions.csv")
	if err := os.WriteFile(p, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p, b.Bytes()
}
func TestCSVImportReviewConfirmationAndReportsSurviveReopen(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		path, original := importFixture(t, importHeader, syntheticImportRow())
		imported, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path, SourceID: "synthetic-current"})
		if err != nil {
			t.Fatal(err)
		}
		if len(imported) != 1 {
			t.Fatal(imported)
		}
		tr := imported[0]
		listed, err := s.ListTrades(ctx)
		if err != nil || len(listed) != 1 {
			t.Fatalf("journal: %v %v", listed, err)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		review, err := s.ReviewMigration(ctx, tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(review.Migration.InferredEssentials) != 2 || len(review.PendingEssentials) < 4 || review.Migration.Fields["NOTES"] != syntheticImportRow()[23] || review.OriginalPlan != nil || review.Migration.ConfirmedAt != nil {
			t.Fatalf("review: %+v", review)
		}
		version, err := s.GetSetupVersion(ctx, tr.SetupVersionID)
		if err != nil || version.Rules != "" {
			t.Fatalf("unknown rules: %+v %v", version, err)
		}
		report, err := s.ReportPerformance(ctx, mtl.ReportFilter{})
		if err != nil || len(report.Groups) != 1 || report.Groups[0].ExcludedCount != 1 || report.Groups[0].EligibleCount != 0 {
			t.Fatalf("pending report: %+v %v", report, err)
		}
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Confirmed venue"})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "Historical account"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		confirmed, err := s.ConfirmMigration(ctx, mtl.ConfirmMigrationRequest{TradeID: tr.ID, AccountID: account.ID, SettlementCurrency: "USDT", Executions: []mtl.MigrationExecution{{Kind: mtl.Entry, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at, ActualFee: decimalPtr("0.1")}}, {Kind: mtl.Exit, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("120"), ExecutedAt: at.Add(time.Hour), ActualFee: decimalPtr("0.2")}}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(confirmed.PendingEssentials) != 0 || confirmed.Migration.ConfirmedAt == nil || len(confirmed.FundingPayments) != 0 {
			t.Fatalf("confirmed %+v", confirmed)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.GetTrade(ctx, tr.ID)
		if err != nil || got.NetProfitLoss.String() != "19.7" || got.Migration.Fields["STOPLOSSPRICE"] != "90" {
			t.Fatalf("reopen: %+v %v", got, err)
		}
		report, err = s.ReportPerformance(ctx, mtl.ReportFilter{})
		if len(report.Completed) != 1 || report.Completed[0].TradeID != tr.ID || report.Completed[0].RMultiple != nil || report.Completed[0].OriginalPlannedRisk != nil {
			t.Fatalf("confirmed migration without historical risk: %+v", report.Completed)
		}
		if err != nil || len(report.Groups) != 1 || report.Groups[0].EligibleCount != 1 || report.Groups[0].ExcludedCount != 0 || report.Groups[0].TotalNetProfit.String() != "19.7" {
			t.Fatalf("confirmed report %+v %v", report, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, original) {
			t.Fatalf("source modified: %v", err)
		}
		again, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path, SourceID: "synthetic-current"})
		if err != nil || len(again) != 1 || again[0].ID != tr.ID || again[0].Migration.ConfirmedAt == nil {
			t.Fatalf("retry: %+v %v", again, err)
		}
	})
}

func TestLegacyCSVPreservesPartialNontradeAndUnknownHistory(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		header := []string{"ID", "SYMBOL", "SIDE", "EXCHANGE", "ENTRYTYPE", "ORDERTYPE", "STRATEGY", "LEVERAGE", "TIMEFRAME", "ENTRYPRICE", "POSITIONMARGIN", "ENTRYFEERATE", "ENTRYMENTALSTATE", "ENTRYIMAGEURL", "OPENEDAT", "EXITPRICE", "EXITFEERATE", "EXITMENTALSTATE", "EXITIMAGEURL", "EXITEDAT", "TAKEPROFITPRICE", "STOPLOSSPRICE", "EXITMARGINPERCENT", "NOTES", "CREATEDAT", "UPDATEDAT"}
		current := syntheticImportRow()
		current[4] = "SELL"
		current[6] = "MARGININOUT"
		current[18] = "25"
		current[19] = ""
		current[23] = "legacy synthetic notes"
		row := make([]string, len(header))
		for i, name := range header {
			for j, key := range importHeader {
				if key == name {
					row[i] = current[j]
				}
			}
		}
		path, original := importFixture(t, header, row)
		trades, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path})
		if err != nil {
			t.Fatal(err)
		}
		tr := trades[0]
		pendingReport, err := s.ReportPerformance(ctx, mtl.ReportFilter{})
		if err != nil || len(pendingReport.Groups) != 0 || len(pendingReport.Ongoing) != 1 {
			t.Fatalf("historical partial exit is ongoing: %+v %v", pendingReport, err)
		}
		if tr.Direction != mtl.Short || len(tr.Executions) != 2 || tr.Executions[1].Quantity.String() != "0.25" || tr.Migration.Format != "csv-legacy" || len(tr.Migration.OriginalFields) != 26 {
			t.Fatalf("legacy: %+v", tr)
		}
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Legacy confirmation"})
		if err != nil {
			t.Fatal(err)
		}
		account, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "main"})
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		req := mtl.ConfirmMigrationRequest{TradeID: tr.ID, AccountID: account.ID, SettlementCurrency: "USDC", Executions: []mtl.MigrationExecution{{Kind: mtl.Entry, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: at, ActualFee: decimalPtr("0")}}, {Kind: mtl.Exit, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("0.25"), Price: mtl.MustDecimal("80"), ExecutedAt: at.Add(time.Hour), ActualFee: decimalPtr("0")}}}}
		if _, err := s.ConfirmMigration(ctx, req); err == nil {
			t.Fatal("nontrade classification silently confirmed")
		}
		req.ConfirmTrade = true
		confirmed, err := s.ConfirmMigration(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if confirmed.CompletedAt != nil || confirmed.RemainingQuantity.String() != "0.75" || confirmed.NetProfitLoss.String() != "5" {
			t.Fatalf("partial: %+v", confirmed)
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.ReportPerformance(ctx, mtl.ReportFilter{})
		if err != nil || len(r.Groups) != 0 || len(r.Ongoing) != 1 {
			t.Fatalf("partial report %+v %v", r, err)
		}
		got, err := s.GetTrade(ctx, tr.ID)
		if err != nil || got.OriginalPlan != nil || got.Migration.Fields["EXITFEERATE"] != "" || got.Migration.Fields["ENTRYTYPE"] != "MARGININOUT" {
			t.Fatalf("unknown history: %+v %v", got, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, original) {
			t.Fatal("legacy source changed", err)
		}
	})
}

func TestMigrationRejectsInvalidBatchesAndIncompleteConfirmationAtomically(t *testing.T) {
	forEachBackend(t, func(t *testing.T, b backend) {
		ctx := context.Background()
		s, err := b.Open()
		if err != nil {
			t.Fatal(err)
		}
		path, original := importFixture(t, importHeader, syntheticImportRow())
		var data bytes.Buffer
		w := csv.NewWriter(&data)
		bad := syntheticImportRow()
		bad[0] = "e85322f7-2c4d-4999-bbf0-25aa510bf552"
		bad[9] = "not-a-price"
		if err := w.WriteAll([][]string{importHeader, syntheticImportRow(), bad}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path, SourceID: "invalid-batch"}); err == nil {
			t.Fatal("malformed batch accepted")
		}
		listed, err := s.ListTrades(ctx)
		if err != nil || len(listed) != 0 {
			t.Fatalf("partial import: %+v %v", listed, err)
		}
		if err := os.WriteFile(path, original, 0600); err != nil {
			t.Fatal(err)
		}
		trades, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path, SourceID: "confirmation"})
		if err != nil {
			t.Fatal(err)
		}
		tr := trades[0]
		ex, err := s.Exchange.Add(ctx, mtl.AddExchangeRequest{Name: "Confirm", EntryFeeRate: decimalPtr("1"), ExitFeeRate: decimalPtr("1")})
		if err != nil {
			t.Fatal(err)
		}
		a, err := s.CreateAccount(ctx, mtl.CreateAccountRequest{ExchangeID: ex.ID, Name: "main"})
		if err != nil {
			t.Fatal(err)
		}
		req := mtl.ConfirmMigrationRequest{TradeID: tr.ID, AccountID: a.ID, SettlementCurrency: "USDT", Executions: []mtl.MigrationExecution{{Kind: mtl.Entry, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("1"), Price: mtl.MustDecimal("100"), ExecutedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}}
		if _, err := s.ConfirmMigration(ctx, req); err == nil {
			t.Fatal("current fee defaults substituted for missing historical fee")
		}
		req.Executions[0].ActualFee = decimalPtr("0")
		req.Executions = append(req.Executions, mtl.MigrationExecution{Kind: mtl.Exit, ExecutionRequest: mtl.ExecutionRequest{Quantity: mtl.MustDecimal("2"), Price: mtl.MustDecimal("120"), ExecutedAt: req.Executions[0].ExecutedAt.Add(time.Hour), ActualFee: decimalPtr("0")}})
		if _, err := s.ConfirmMigration(ctx, req); err == nil {
			t.Fatal("overexit confirmed")
		}
		s, err = b.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.ReviewMigration(ctx, tr.ID)
		if err != nil || got.Migration.ConfirmedAt != nil || len(got.PendingEssentials) == 0 {
			t.Fatalf("failed confirmation mutated data: %+v %v", got, err)
		}
		changed := syntheticImportRow()
		changed[23] = "modified source"
		var modified bytes.Buffer
		cw := csv.NewWriter(&modified)
		if err := cw.WriteAll([][]string{importHeader, changed}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, modified.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: path, SourceID: "confirmation"}); err == nil {
			t.Fatal("changed source overwritten")
		}
		source, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(source, modified.Bytes()) {
			t.Fatalf("source modified by rejection: %v", err)
		}
		missing := filepath.Join(t.TempDir(), "missing.csv")
		if _, err := s.ImportCSV(ctx, mtl.CSVImportRequest{Path: missing}); err == nil {
			t.Fatal("missing source accepted")
		}
		if _, err := os.Stat(missing); !os.IsNotExist(err) {
			t.Fatal("source created", err)
		}
	})
}
