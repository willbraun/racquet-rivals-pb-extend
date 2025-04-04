package paddle_webhooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterAllPaddleWebhooks(app core.App) {
	RegisterDrawEntryTransactionCompletedHook(app)
	RegisterSubscriptionActivatedHook(app)
}
