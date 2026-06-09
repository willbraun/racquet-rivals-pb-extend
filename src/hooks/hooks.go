package hooks

import (
	"pocketbase_extend/src/hooks/paddle_webhooks"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterAllHooks(app core.App) {
	RegisterAccessHook(app)
	RegisterPredictionCloseHook(app)
	RegisterAwardPointsHook(app)
	RegisterEndDrawHook(app)
	paddle_webhooks.RegisterAllPaddleWebhooks(app)
}
