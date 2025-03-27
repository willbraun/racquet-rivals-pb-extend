package paddle_webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterDrawEntryTransactionCompletedHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/webhook/draw-entry-transaction-completed", func(e *core.RequestEvent) error {
			// Get the request body
			defer e.Request.Body.Close()
			bodyBytes, err := io.ReadAll(e.Request.Body)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Failed to read request body",
					"details": err.Error(),
				})
			}

			var requestBody PaddleTransactionCompleted
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"error":   "Invalid JSON format",
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
				case "pro_01jpkgny45pjrxh3nnb7nck2zv":
					drawTypes = append(drawTypes, "Men")
				case "pro_01jpkhe7h0se4fcr3bpf3cha1x":
					drawTypes = append(drawTypes, "Women")
				case "pro_01jpkhhgn8qzpg0ry0yvgyqw8d":
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
