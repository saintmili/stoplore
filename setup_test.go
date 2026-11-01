package mtl_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"testing"

	mtl "github.com/saintmili/mtl"
)

func TestSetupRulesRemainAvailableAfterRevisionAndReopen(t *testing.T) {
	forEachBackend(t, func(t *testing.T, backend backend) {
		ctx := context.Background()
		service, err := backend.Open()
		if err != nil {
			t.Fatal(err)
		}
		setup, err := service.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Breakout", Rules: "Enter after daily close above resistance"})
		if err != nil {
			t.Fatal(err)
		}
		if setup.Name != "Breakout" || len(setup.Versions) != 1 || setup.Versions[0].Version != 1 {
			t.Fatalf("unexpected initial setup: %+v", setup)
		}
		original := setup.Versions[0]
		revised, err := service.ReviseSetup(ctx, mtl.ReviseSetupRequest{SetupID: setup.ID, Rules: "Enter after daily close and retest"})
		if err != nil {
			t.Fatal(err)
		}
		if revised.ID == original.ID || revised.SetupID != setup.ID || revised.Version != 2 {
			t.Fatalf("revision lacks distinct identity: %+v", revised)
		}
		reopened, err := backend.Open()
		if err != nil {
			t.Fatal(err)
		}
		saved, err := reopened.GetSetup(ctx, setup.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(saved.Versions) != 2 || saved.Versions[0].Rules != "Enter after daily close above resistance" || saved.Versions[1].Rules != "Enter after daily close and retest" {
			t.Fatalf("setup rules lost: %+v", saved)
		}
		historical, err := reopened.GetSetupVersion(ctx, original.ID)
		if err != nil {
			t.Fatal(err)
		}
		if historical.ID != original.ID || historical.Rules != "Enter after daily close above resistance" || historical.CreatedAt.IsZero() {
			t.Fatalf("historical rules lost: %+v", historical)
		}
	})
}

func TestListSetupsKeepsIndependentNamedRules(t *testing.T) {
	forEachBackend(t, func(t *testing.T, backend backend) {
		ctx := context.Background()
		service, err := backend.Open()
		if err != nil {
			t.Fatal(err)
		}
		first, err := service.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Trend", Rules: "Trade with daily trend"})
		if err != nil {
			t.Fatal(err)
		}
		second, err := service.CreateSetup(ctx, mtl.CreateSetupRequest{Name: "Breakout", Rules: "Wait for range break"})
		if err != nil {
			t.Fatal(err)
		}
		setups, err := service.ListSetups(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(setups) != 2 || setups[0].ID != second.ID || setups[1].ID != first.ID {
			t.Fatalf("unexpected named setups: %+v", setups)
		}
	})
}

func TestSetupRequestsReturnMeaningfulErrors(t *testing.T) {
	forEachBackend(t, func(t *testing.T, backend backend) {
		ctx := context.Background()
		service, err := backend.Open()
		if err != nil {
			t.Fatal(err)
		}
		for _, request := range []mtl.CreateSetupRequest{{Name: " ", Rules: "Wait"}, {Name: "Breakout", Rules: " "}} {
			if _, err := service.CreateSetup(ctx, request); !errors.Is(err, mtl.ErrInvalidRequest) {
				t.Fatalf("invalid setup: %v", err)
			}
		}
		missing := uuid.New()
		if _, err := service.GetSetup(ctx, missing); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing setup: %v", err)
		}
		if _, err := service.GetSetupVersion(ctx, missing); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("missing setup version: %v", err)
		}
		if _, err := service.ReviseSetup(ctx, mtl.ReviseSetupRequest{SetupID: missing, Rules: "Wait"}); !errors.Is(err, mtl.ErrNotFound) {
			t.Fatalf("revise missing setup: %v", err)
		}
		if _, err := service.ReviseSetup(ctx, mtl.ReviseSetupRequest{SetupID: missing, Rules: " "}); !errors.Is(err, mtl.ErrInvalidRequest) {
			t.Fatalf("empty revised rules: %v", err)
		}
		setups, err := service.ListSetups(ctx)
		if err != nil || len(setups) != 0 {
			t.Fatalf("invalid requests persisted setups: %+v %v", setups, err)
		}
	})
}
