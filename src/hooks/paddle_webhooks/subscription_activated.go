package paddle_webhooks

import (
	"database/sql"
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
		route := "/webhook/subscription-activated"
		se.Router.POST(route, func(e *core.RequestEvent) error {
			// Ensure the request is from Paddle
			verifyWebhook := os.Getenv("SKIP_WEBHOOK_VERIFICATION") != "true"
			if verifyWebhook {
				webhookVerifier := paddle.NewWebhookVerifier(os.Getenv("SUBSCRIPTION_ACTIVATED_WEBHOOK_SECRET_KEY"))
				_, err := webhookVerifier.Verify(e.Request)
				if err != nil {
					return HandleWebhookError(WebhookErrorContext{
						App:        app,
						Event:      e,
						StatusCode: http.StatusBadRequest,
						Route:      route,
						Message:    "Invalid webhook signature",
						Error:      err,
					})
				}
			}

			// Get the request body
			defer e.Request.Body.Close()
			bodyBytes, err := io.ReadAll(e.Request.Body)
			if err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusBadRequest,
					Route:            route,
					Message:          "Failed to read request body",
					RequestBodyBytes: nil,
					Error:            err,
				})
			}

			var requestBody PaddleSubscriptionActivated
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusBadRequest,
					Route:            route,
					Message:          "Invalid JSON format",
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			// Validate the webhook payload
			if err := validateSubscriptionActivatedPayload(requestBody); err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusBadRequest,
					Route:            route,
					Message:          fmt.Sprintf("Invalid webhook payload format: %s", err.Error()),
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			// Get the user who activated the subscription
			userId := requestBody.Data.CustomData.UserID
			_, err = app.FindRecordById("user", userId)
			if err != nil {
				if err == sql.ErrNoRows {
					return HandleWebhookError(WebhookErrorContext{
						App:              app,
						Event:            e,
						StatusCode:       http.StatusNotFound,
						Route:            route,
						Message:          fmt.Sprintf("User with ID '%s' not found", userId),
						RequestBodyBytes: bodyBytes,
						Error:            err,
					})
				}

				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          fmt.Sprintf("Internal error finding user '%s'", userId),
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			// If user already has an active subscription, do not create a new one
			filter := fmt.Sprintf(`user_id="%s"`, userId)
			existingSubscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
			if err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          fmt.Sprintf("Internal error checking existing subscriptions for user '%s'", userId),
					RequestBodyBytes: bodyBytes,
					Error:            err,
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
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          "Internal error finding subscription collection",
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			record := core.NewRecord(subscription)
			record.Set("user_id", userId)
			record.Set("paddle_subscription_id", requestBody.Data.ID)
			record.Set("status", requestBody.Data.Status)
			record.Set("current_billing_period_start", requestBody.Data.CurrentBillingPeriod.StartsAt)
			record.Set("current_billing_period_end", requestBody.Data.CurrentBillingPeriod.EndsAt)

			if err := app.Save(record); err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          fmt.Sprintf("Failed to save new subscription record for user_id '%s' and paddle_subscription_id: '%s'", userId, requestBody.Data.ID),
					RequestBodyBytes: bodyBytes,
					Error:            err,
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
