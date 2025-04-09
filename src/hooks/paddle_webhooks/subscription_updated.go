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

// Note: Paddle webhooks are always sent as POST requests, even for updates

func RegisterSubscriptionUpdatedHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		route := "/webhook/subscription-updated"
		se.Router.POST(route, func(e *core.RequestEvent) error {
			// Ensure the request is from Paddle
			verifyWebhook := os.Getenv("SKIP_WEBHOOK_VERIFICATION") != "true"
			if verifyWebhook {
				webhookVerifier := paddle.NewWebhookVerifier(os.Getenv("SUBSCRIPTION_UPDATED_WEBHOOK_SECRET_KEY"))
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

			var requestBody PaddleSubscriptionUpdated
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
			if err := validateSubscriptionUpdatedPayload(requestBody); err != nil {
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

			// Get the subscription record
			paddleSubscriptionId := requestBody.Data.ID
			filter := fmt.Sprintf(`paddle_subscription_id="%s"`, paddleSubscriptionId)
			record, err := app.FindFirstRecordByFilter("subscription", filter)
			if err != nil {
				if err == sql.ErrNoRows {
					return HandleWebhookError(WebhookErrorContext{
						App:              app,
						Event:            e,
						StatusCode:       http.StatusNotFound,
						Route:            route,
						Message:          fmt.Sprintf("Subscription with paddle_subscription_id '%s' not found", paddleSubscriptionId),
						RequestBodyBytes: bodyBytes,
						Error:            err,
					})
				}

				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          fmt.Sprintf("Internal error finding subscription record for paddle_subscription_id '%s'", paddleSubscriptionId),
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			userId := record.GetString("user_id")

			// Update subscription record
			record.Set("status", requestBody.Data.Status)
			record.Set("current_billing_period_start", requestBody.Data.CurrentBillingPeriod.StartsAt)
			record.Set("current_billing_period_end", requestBody.Data.CurrentBillingPeriod.EndsAt)

			if err := app.Save(record); err != nil {
				return HandleWebhookError(WebhookErrorContext{
					App:              app,
					Event:            e,
					StatusCode:       http.StatusInternalServerError,
					Route:            route,
					Message:          fmt.Sprintf("Failed to save subscription record update for user_id '%s' and paddle_subscription_id '%s'", userId, requestBody.Data.ID),
					RequestBodyBytes: bodyBytes,
					Error:            err,
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"message": "Subscription updated successfully",
			})
		})

		return se.Next()
	})
}

func validateSubscriptionUpdatedPayload(payload PaddleSubscriptionUpdated) error {
	if payload.EventID == "" {
		return fmt.Errorf("missing event_id")
	}

	if payload.EventType != "subscription.updated" {
		return fmt.Errorf("invalid event_type. Expected subscription.updated, got %s", payload.EventType)
	}

	if len(payload.Data.Items) == 0 {
		return fmt.Errorf("no items in transaction")
	}

	if payload.Data.Items[0].Product.ID != "pro_01jpkhsd61k1acva107vz6dj0v" {
		return fmt.Errorf("invalid product ID. Expected pro_01jpkhsd61k1acva107vz6dj0v, got %s", payload.Data.Items[0].Product.ID)
	}

	return nil
}
