package yacypeerassembly

import (
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberlog"
)

func ConfigureLogging(getenv func(string) string) error {
	handler, err := applog.HandlerFor(getenv)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(memberlog.NewHandler(handler)))
	return nil
}
