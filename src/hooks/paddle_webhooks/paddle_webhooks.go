package paddle_webhooks

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

var productIDs map[string]string

func LoadProductIDs() {
	productIDs = map[string]string{
		"Men":          os.Getenv("MENS_PRODUCT_ID"),
		"Women":        os.Getenv("WOMENS_PRODUCT_ID"),
		"Both":         os.Getenv("BOTH_PRODUCT_ID"),
		"Subscription": os.Getenv("SUBSCRIPTION_PRODUCT_ID"),
	}
}

func getProductID(productType string) string {
	id, ok := productIDs[productType]
	if !ok {
		log.Panicf("unknown product type %q", productType)
	}
	return id
}

func RegisterAllPaddleWebhooks(app core.App) {
	RegisterTransactionCompletedHook(app)
	RegisterSubscriptionActivatedHook(app)
	RegisterSubscriptionUpdatedHook(app)
}
