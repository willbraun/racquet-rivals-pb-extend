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

var productIDs = map[string]string{
	"Men":   "pro_01jpkgny45pjrxh3nnb7nck2zv",
	"Women": "pro_01jpkhe7h0se4fcr3bpf3cha1x",
	"Both":  "pro_01jpkhhgn8qzpg0ry0yvgyqw8d",
}

func RegisterDrawEntryTransactionCompletedHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/webhook/draw-entry-transaction-completed", func(e *core.RequestEvent) error {
			// Ensure the request is from Paddle
			verifyWebhook := os.Getenv("SKIP_WEBHOOK_VERIFICATION") != "true"
			if verifyWebhook {
				webhookVerifier := paddle.NewWebhookVerifier(os.Getenv("DRAW_ENTRY_WEBHOOK_SECRET_KEY"))
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

			var requestBody PaddleDrawEntryTransaction
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Invalid JSON format",
					"details": err.Error(),
				})
			}

			// Validate the webhook payload
			if err := validateDrawEntryTransactionCompletedPayload(requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Invalid webhook payload format",
					"details": err.Error(),
				})
			}

			// Get the user who made the purchase
			userId := requestBody.Data.CustomData.UserID
			_, err = app.FindRecordById("user", userId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]any{
					"error":   fmt.Sprintf("User with ID %s not found", userId),
					"details": err.Error(),
				})
			}

			// Get the product purchased (Men, Women, or Both)
			transactionItems := requestBody.Data.Items

			if len(transactionItems) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error": "No valid products found in transaction",
				})
			}

			// Determine which draws to give access to
			var drawTypes []string
		items:
			for _, item := range transactionItems {
				switch item.Price.ProductID {
				case productIDs["Men"]:
					drawTypes = append(drawTypes, "Men")
				case productIDs["Women"]:
					drawTypes = append(drawTypes, "Women")
				case productIDs["Both"]:
					// "Both" product overrides individual products
					drawTypes = []string{"Men", "Women"}
					break items
				}
			}

			// Create the user_draw_entry records
			for _, drawType := range drawTypes {
				switch drawType {
				case "Men":
					if requestBody.Data.CustomData.MensDrawID == nil {
						return e.JSON(http.StatusBadRequest, map[string]any{
							"error": "Men's draw ID is required for men's product",
						})
					}

					mensDrawId := *requestBody.Data.CustomData.MensDrawID
					_, err = app.FindRecordById("draw", mensDrawId)
					if err != nil {
						return e.JSON(http.StatusNotFound, map[string]any{
							"error":   fmt.Sprintf("Men's draw with ID %s not found", mensDrawId),
							"details": err.Error(),
						})
					}

					if err := createUserDrawEntryRecord(app, e, userId, mensDrawId); err != nil {
						return err
					}

				case "Women":
					if requestBody.Data.CustomData.WomensDrawID == nil {
						return e.JSON(http.StatusBadRequest, map[string]any{
							"error": "Women's draw ID is required for women's product",
						})
					}

					womensDrawId := *requestBody.Data.CustomData.WomensDrawID
					_, err = app.FindRecordById("draw", womensDrawId)
					if err != nil {
						return e.JSON(http.StatusNotFound, map[string]any{
							"error":   fmt.Sprintf("Women's draw with ID %s not found", womensDrawId),
							"details": err.Error(),
						})
					}

					if err := createUserDrawEntryRecord(app, e, userId, womensDrawId); err != nil {
						return err
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]any{
				"message": "Draw entry added successfully",
			})
		})

		return se.Next()
	})
}

func createUserDrawEntryRecord(app core.App, e *core.RequestEvent, userId string, drawId string) error {
	// If entry already exists, consider this a success (idempotent)
	filter := fmt.Sprintf(`user_id="%s"&&draw_id="%s"`, userId, drawId)
	existingEntries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "Failed to check for existing entries",
			"details": err.Error(),
		})
	}

	if len(existingEntries) > 0 {
		return nil
	}

	// Create new user_draw_entry record
	userDrawEntry, err := app.FindCollectionByNameOrId("user_draw_entry")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]any{
			"error":   "Failed to find user_draw_entry collection",
			"details": err.Error(),
		})
	}

	record := core.NewRecord(userDrawEntry)
	record.Set("user_id", userId)
	record.Set("draw_id", drawId)

	if err := app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]any{
			"error":   fmt.Sprintf("Failed to save user_draw_entry record with user_id: %s and draw_id: %s", userId, drawId),
			"details": err.Error(),
		})
	}

	return nil
}

func validateDrawEntryTransactionCompletedPayload(payload PaddleDrawEntryTransaction) error {
	if payload.EventID == "" {
		return fmt.Errorf("missing event_id")
	}

	if payload.EventType != "transaction.completed" {
		return fmt.Errorf("invalid event_type. Expected transaction.completed, got %s", payload.EventType)
	}

	if payload.Data.CustomData.UserID == "" {
		return fmt.Errorf("missing user_id in custom_data")
	}

	if len(payload.Data.Items) == 0 {
		return fmt.Errorf("no items in transaction")
	}

	validProductFound := false
	validProductIDs := map[string]bool{
		productIDs["Men"]:   true,
		productIDs["Women"]: true,
		productIDs["Both"]:  true,
	}

	for _, item := range payload.Data.Items {
		if validProductIDs[item.Price.ProductID] {
			validProductFound = true
		}

		if item.Price.ProductID == productIDs["Men"] && payload.Data.CustomData.MensDrawID == nil {
			return fmt.Errorf("men's draw entry must also have mens_draw_id in custom_data")
		}

		if item.Price.ProductID == productIDs["Women"] && payload.Data.CustomData.WomensDrawID == nil {
			return fmt.Errorf("women's draw entry must also have womens_draw_id in custom_data")
		}

		if item.Price.ProductID == productIDs["Both"] && (payload.Data.CustomData.MensDrawID == nil || payload.Data.CustomData.WomensDrawID == nil) {
			return fmt.Errorf("men's and women's joint entry must also have mens_draw_id and womens_draw_id in custom_data")
		}
	}

	if !validProductFound {
		return fmt.Errorf("no valid products found")
	}

	return nil
}
