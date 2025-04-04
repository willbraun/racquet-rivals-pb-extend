package paddle_webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	paddle "github.com/PaddleHQ/paddle-go-sdk/v3"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterSubscriptionActivatedHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/webhook/subscription-activated", func(e *core.RequestEvent) error {
			// Ensure the request is from Paddle
			verifyWebhook := os.Getenv("SKIP_WEBHOOK_VERIFICATION") != "true"
			if verifyWebhook {
				webhookVerifier := paddle.NewWebhookVerifier(os.Getenv("SUBSCRIPTION_ACTIVATED_WEBHOOK_SECRET_KEY"))
				_, err := webhookVerifier.Verify(e.Request)
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]any{
						"error":   "Invalid webhook signature",
						"details": err.Error(),
					})
				}
			}

			// Get the request body
			defer e.Request.Body.Close()
			bodyBytes, err := io.ReadAll(e.Request.Body)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Failed to read request body",
					"details": err.Error(),
				})
			}

			var requestBody PaddleSubscriptionActivated
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Invalid JSON format",
					"details": err.Error(),
				})
			}

			// Validate the webhook payload
			if err := validateSubscriptionActivatedPayload(requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Invalid webhook payload format",
					"details": err.Error(),
				})
			}

			// Get the user who activated the subscription
			userId := requestBody.Data.CustomData.UserID
			_, err = app.FindRecordById("user", userId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]any{
					"error":   fmt.Sprintf("User with ID %s not found", userId),
					"details": err.Error(),
				})
			}

			// If user already has an active subscription, do not create a new one
			filter := fmt.Sprintf(`user_id="%s"`, userId)
			existingSubscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]any{
					"error":   "Failed to check for existing entries",
					"details": err.Error(),
				})
			}

			if len(existingSubscriptions) > 0 {
				for _, sub := range existingSubscriptions {
					if sub.GetString("status") == "active" {
						return e.JSON(http.StatusOK, map[string]any{
							"message": "User already has an active subscription, no new subscription created",
						})
					}
				}
			}

			// Create new subscription
			subscription, err := app.FindCollectionByNameOrId("subscription")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]any{
					"error":   "Failed to find subscription collection",
					"details": err.Error(),
				})
			}

			record := core.NewRecord(subscription)
			record.Set("user_id", userId)
			record.Set("paddle_subscription_id", requestBody.Data.ID)
			record.Set("status", requestBody.Data.Status)
			record.Set("current_billing_period_start", requestBody.Data.CurrentBillingPeriod.StartsAt)
			record.Set("current_billing_period_end", requestBody.Data.CurrentBillingPeriod.EndsAt)

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]any{
					"error":   fmt.Sprintf("Failed to save subscription record for user_id: %s and paddle_subscription_id: %s", userId, requestBody.Data.ID),
					"details": err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"message": "Subscription activated successfully",
			})
		})

		return se.Next()
	})
}

func validateSubscriptionActivatedPayload(payload PaddleSubscriptionActivated) error {
	if payload.EventID == "" {
		return fmt.Errorf("missing event_id")
	}

	if payload.EventType != "subscription.activated" {
		return fmt.Errorf("invalid event_type. Expected subscription.activated, got %s", payload.EventType)
	}

	if payload.Data.CustomData.UserID == "" {
		return fmt.Errorf("missing user_id in custom_data")
	}

	if len(payload.Data.Items) == 0 {
		return fmt.Errorf("no items in transaction")
	}

	if payload.Data.Items[0].Product.ID != "pro_01jpkhsd61k1acva107vz6dj0v" {
		return fmt.Errorf("invalid product ID. Expected pro_01jpkhsd61k1acva107vz6dj0v, got %s", payload.Data.Items[0].Product.ID)
	}

	if payload.Data.Status != "active" {
		return fmt.Errorf("subscription status is not active: %s", payload.Data.Status)
	}

	if payload.Data.CurrentBillingPeriod.EndsAt.IsZero() || payload.Data.CurrentBillingPeriod.StartsAt.IsZero() {
		return fmt.Errorf("missing billing period dates")
	}

	return nil
}
