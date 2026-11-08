// Package companionsettings keeps the settings of the companions of the peer in one file of a
// directory: whether paired companions fetch pages, on until the operator turns it off, and
// whether the peer fetches pages itself while no companion is connected. It tells its observers
// each change, the kept fetching first.
package companionsettings

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
	fileName          = "companion-settings.json"
	draftFileName     = fileName + ".draft"
	directoryFileMode = 0o700
	fileMode          = 0o600
)

type Fetching struct {
	CompanionsFetch bool `json:"companionsFetch"`
	PeerFillsIn     bool `json:"peerFillsIn"`
}

func (f Fetching) OnlyCompanionsFetch() bool {
	return f.CompanionsFetch && !f.PeerFillsIn
}

type Observer interface {
	FetchingChanged(ctx context.Context, fetching Fetching)
}

type Keeper struct {
	root      *os.Root
	observers []Observer
	fetching  Fetching
	mu        sync.Mutex
}

func Open(ctx context.Context, directory string, observers ...Observer) (*Keeper, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the companion settings directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the companion settings directory: %w", err)
	}
	fetching, err := keptFetchingIn(root)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	keeper := &Keeper{root: root, observers: observers, fetching: fetching}
	keeper.tell(ctx)
	return keeper, nil
}

func keptFetchingIn(root *os.Root) (Fetching, error) {
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return Fetching{CompanionsFetch: true}, nil
	}
	if err != nil {
		return Fetching{}, fmt.Errorf("read the companion settings: %w", err)
	}
	var fetching Fetching
	if err := json.Unmarshal(kept, &fetching); err != nil {
		return Fetching{}, fmt.Errorf("read the companion settings: %w", err)
	}
	return fetching, nil
}

func (k *Keeper) Fetching() Fetching {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.fetching
}

func (k *Keeper) ChangeFetching(ctx context.Context, fetching Fetching) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.fetching = fetching
	k.keep(ctx)
	k.tell(ctx)
}

func (k *Keeper) keep(ctx context.Context) {
	encoded, err := json.MarshalIndent(k.fetching, "", "  ")
	if err == nil {
		err = k.root.WriteFile(draftFileName, append(encoded, '\n'), fileMode)
	}
	if err == nil {
		err = k.root.Rename(draftFileName, fileName)
	}
	if err != nil {
		slog.ErrorContext(ctx, "companion settings not kept", slog.Any("error", err))
	}
}

func (k *Keeper) tell(ctx context.Context) {
	for _, observer := range k.observers {
		observer.FetchingChanged(ctx, k.fetching)
	}
}

func (k *Keeper) Close() {
	if err := k.root.Close(); err != nil {
		slog.WarnContext(context.Background(), "companion settings directory not closed",
			slog.Any("error", err),
		)
	}
}
