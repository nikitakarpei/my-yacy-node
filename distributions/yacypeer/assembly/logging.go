package yacypeerassembly

import (
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberlog"
)

func ConfigureLogging(getenv func(string) string) error {
	if err := applog.Configure(getenv); err != nil {
		return err
	}
	slog.SetDefault(slog.New(memberlog.NewHandler(slog.Default().Handler())))
	return nil
}
