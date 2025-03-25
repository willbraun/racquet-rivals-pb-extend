package paddle_webhooks

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterDrawEntryTransactionCompletedHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/webhook/draw-entry-transaction-completed", func(e *core.RequestEvent) error {
			bodyBytes, err := io.ReadAll(e.Request.Body)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Failed to read request body",
				})
			}
			defer e.Request.Body.Close()

			var requestBody PaddleTransactionCompleted
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":   "Invalid JSON format",
					"details": err.Error(),
				})
			}

			// TODO: Implement the logic for this webhook
			return e.JSON(http.StatusOK, requestBody)

			// if this is not transaction.completed event, return early, status 400

			// Get the user who made the purchase, by user_id in custom_data
			// If no user or user not found, throw error (and send me email?)

			// Parse the request body to get the product purchased (Men, Women, or Both)
			// if product not recognized, throw error

			// Get the active draw(s) of that product type

			// Create a new draw entry for the user and the active draw(s)
			// if this errors, throw error

			// Return a success response
		})

		return se.Next()
	})
}
