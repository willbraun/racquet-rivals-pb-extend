package hooks

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Check if a given user has access to participate in a given draw
func RegisterAccessHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/access/{user_id}/{draw_id}", func(e *core.RequestEvent) error {
			userId := strings.Trim(e.Request.PathValue("user_id"), " ")
			drawId := strings.Trim(e.Request.PathValue("draw_id"), " ")

			if userId == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Must provide user_id",
				})
			}

			if drawId == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Must provide draw_id",
				})
			}

			user, err := app.FindRecordById("user", userId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]any{
					"error":   "User not found",
					"details": err.Error(),
				})
			}

			requestedUsername := user.GetString("username")

			if requestedUsername != e.Auth.GetString("username") {
				return e.JSON(http.StatusForbidden, map[string]string{
					"error": fmt.Sprintf("You don't have permission to access %s's data", requestedUsername),
				})
			}

			_, err = app.FindRecordById("draw", drawId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]any{
					"error":   "Draw not found",
					"details": err.Error(),
				})
			}

			// Check if user is grandfathered in
			if user.GetBool("grandfathered") {
				return e.JSON(http.StatusOK, map[string]any{
					"hasAccess": true,
				})
			}

			// Check if user has active or past due subscription
			// If payment fails, the status will be "past_due" and they still have access
			// Paddle will enter the dunning process to recover the payment automatically
			// After 30 days if the payment is not recovered, the status will change to "canceled"
			subscriptionFilter := fmt.Sprintf(`user_id="%s"`, userId)
			subscriptions, err := app.FindRecordsByFilter("subscription", subscriptionFilter, "", 1, 0)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]any{
					"error":   "Error checking subscription access",
					"details": err.Error(),
				})
			}

			if len(subscriptions) > 1 {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Multiple subscriptions found for user",
				})
			}

			if len(subscriptions) == 1 {
				subscription := subscriptions[0]
				status := subscription.GetString("status")
				if status == "active" || status == "past_due" {
					return e.JSON(http.StatusOK, map[string]any{
						"hasAccess": true,
					})
				}
			}

			// Check if user has specifically paid for this draw
			drawEntryFilter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, drawId)
			entries, err := app.FindRecordsByFilter("user_draw_entry", drawEntryFilter, "", 1, 0)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]any{
					"error":   "Error checking draw access",
					"details": err.Error(),
				})
			}

			if len(entries) > 0 {
				return e.JSON(http.StatusOK, map[string]bool{
					"hasAccess": true,
				})
			}

			// If none of the access conditions were met
			return e.JSON(http.StatusOK, map[string]any{
				"hasAccess": false,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
