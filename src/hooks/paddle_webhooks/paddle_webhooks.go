package paddle_webhooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterAllPaddleWebhooks(app core.App) {
	RegisterTransactionCompletedHook(app)
	RegisterSubscriptionActivatedHook(app)
	RegisterSubscriptionUpdatedHook(app)
}
