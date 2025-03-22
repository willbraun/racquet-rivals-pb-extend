package hooks

import "github.com/pocketbase/pocketbase/core"

func RegisterAllHooks(app core.App) {
	RegisterAccessHook(app)
	RegisterPredictionCloseHook(app)
	RegisterAwardPointsHook(app)
}
