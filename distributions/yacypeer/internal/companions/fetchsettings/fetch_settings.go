// Package fetchsettings keeps who fetches pages for the peer in one file of a directory: whether
// paired companions fetch pages, on until the operator turns it off, and whether the peer fetches
// pages itself while no companion is connected. It tells its observers each choice, the kept one
// first.
package fetchsettings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sync"
)

const (
	fileName          = "fetch-settings.json"
	draftFileName     = fileName + ".draft"
	directoryFileMode = 0o700
	fileMode          = 0o600
)

type Choice struct {
	CompanionsFetch bool `json:"companionsFetch"`
	PeerFillsIn     bool `json:"peerFillsIn"`
}

type Observer interface {
	ChoiceMade(ctx context.Context, choice Choice)
}

type Settings struct {
	root      *os.Root
	observers []Observer
	choice    Choice
	mu        sync.Mutex
}

func Open(ctx context.Context, directory string, observers ...Observer) (*Settings, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the fetch settings directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the fetch settings directory: %w", err)
	}
	choice, err := keptChoiceIn(root)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	settings := &Settings{root: root, observers: observers, choice: choice}
	settings.tell(ctx)
	return settings, nil
}

func keptChoiceIn(root *os.Root) (Choice, error) {
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return Choice{CompanionsFetch: true}, nil
	}
	if err != nil {
		return Choice{}, fmt.Errorf("read the fetch settings: %w", err)
	}
	var choice Choice
	if err := json.Unmarshal(kept, &choice); err != nil {
		return Choice{}, fmt.Errorf("read the fetch settings: %w", err)
	}
	return choice, nil
}

func (s *Settings) Choice() Choice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.choice
}

func (s *Settings) Choose(ctx context.Context, choice Choice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.choice = choice
	s.keep(ctx)
	s.tell(ctx)
}

func (s *Settings) keep(ctx context.Context) {
	encoded, err := json.MarshalIndent(s.choice, "", "  ")
	if err == nil {
		err = s.root.WriteFile(draftFileName, append(encoded, '\n'), fileMode)
	}
	if err == nil {
		err = s.root.Rename(draftFileName, fileName)
	}
	if err != nil {
		slog.ErrorContext(ctx, "fetch settings not kept", slog.Any("error", err))
	}
}

func (s *Settings) tell(ctx context.Context) {
	for _, observer := range s.observers {
		observer.ChoiceMade(ctx, s.choice)
	}
}

func (s *Settings) Close() error {
	if err := s.root.Close(); err != nil {
		return fmt.Errorf("close the fetch settings directory: %w", err)
	}
	return nil
}
