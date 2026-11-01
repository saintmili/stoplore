package stoplore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Setup struct {
	ID       uuid.UUID
	Name     string
	Versions []SetupVersion
}

type SetupVersion struct {
	ID        uuid.UUID
	SetupID   uuid.UUID
	Version   int
	Rules     string
	CreatedAt time.Time
}

type CreateSetupRequest struct {
	Name  string
	Rules string
}

type ReviseSetupRequest struct {
	SetupID uuid.UUID
	Rules   string
}

func (s *Service) CreateSetup(ctx context.Context, request CreateSetupRequest) (Setup, error) {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Rules) == "" {
		return Setup{}, fmt.Errorf("%w: setup name and rules are required", ErrInvalidRequest)
	}
	setup := Setup{ID: uuid.New(), Name: request.Name}
	setup.Versions = []SetupVersion{{ID: uuid.New(), SetupID: setup.ID, Version: 1, Rules: request.Rules, CreatedAt: time.Now().UTC()}}
	err := s.update(ctx, func(state *journalState) error {
		setups := map[string]Setup{}
		if err := state.get("setups", &setups); err != nil {
			return err
		}
		setups[setup.ID.String()] = setup
		return state.put("setups", setups)
	})
	if err != nil {
		return Setup{}, err
	}
	return setup, nil
}

func (s *Service) ReviseSetup(ctx context.Context, request ReviseSetupRequest) (SetupVersion, error) {
	if request.SetupID == uuid.Nil || strings.TrimSpace(request.Rules) == "" {
		return SetupVersion{}, fmt.Errorf("%w: setup ID and revised rules are required", ErrInvalidRequest)
	}
	var version SetupVersion
	err := s.update(ctx, func(state *journalState) error {
		setups := map[string]Setup{}
		if err := state.get("setups", &setups); err != nil {
			return err
		}
		setup, ok := setups[request.SetupID.String()]
		if !ok {
			return fmt.Errorf("%w: setup %s", ErrNotFound, request.SetupID)
		}
		version = SetupVersion{ID: uuid.New(), SetupID: setup.ID, Version: len(setup.Versions) + 1, Rules: request.Rules, CreatedAt: time.Now().UTC()}
		setup.Versions = append(setup.Versions, version)
		setups[setup.ID.String()] = setup
		return state.put("setups", setups)
	})
	if err != nil {
		return SetupVersion{}, err
	}
	return version, nil
}

func (s *Service) GetSetup(ctx context.Context, id uuid.UUID) (Setup, error) {
	state, err := s.read(ctx)
	if err != nil {
		return Setup{}, err
	}
	setups := map[string]Setup{}
	if err := state.get("setups", &setups); err != nil {
		return Setup{}, err
	}
	setup, ok := setups[id.String()]
	if !ok {
		return Setup{}, fmt.Errorf("%w: setup %s", ErrNotFound, id)
	}
	return setup, nil
}

func (s *Service) GetSetupVersion(ctx context.Context, id uuid.UUID) (SetupVersion, error) {
	state, err := s.read(ctx)
	if err != nil {
		return SetupVersion{}, err
	}
	setups := map[string]Setup{}
	if err := state.get("setups", &setups); err != nil {
		return SetupVersion{}, err
	}
	for _, setup := range setups {
		for _, version := range setup.Versions {
			if version.ID == id {
				return version, nil
			}
		}
	}
	return SetupVersion{}, fmt.Errorf("%w: setup version %s", ErrNotFound, id)
}

func (s *Service) ListSetups(ctx context.Context) ([]Setup, error) {
	state, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	setups := map[string]Setup{}
	if err := state.get("setups", &setups); err != nil {
		return nil, err
	}
	result := make([]Setup, 0, len(setups))
	for _, setup := range setups {
		result = append(result, setup)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID.String() < result[j].ID.String()
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}
