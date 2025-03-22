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
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Must provide user_id",
				})
			}

			if drawId == "" {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Must provide draw_id",
				})
			}

			user, err := app.FindRecordById("user", userId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "User not found",
				})
			}

			requestedUsername := user.GetString("username")

			if requestedUsername != e.Auth.GetString("username") {
				return e.JSON(http.StatusForbidden, map[string]interface{}{
					"error": fmt.Sprintf("You don't have permission to access %s's data", requestedUsername),
				})
			}

			draw, err := app.FindRecordById("draw", drawId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Draw not found",
				})
			}

			// Check if user is grandfathered in
			if user.GetBool("grandfathered") {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"hasAccess": true,
				})
			}

			// Check if user has active subscription covering the draw date
			drawStartDate := draw.GetDateTime("start_date")
			subscriptionStartDate := user.GetDateTime("subscription_start_date")
			subscriptionEndDate := user.GetDateTime("subscription_end_date")

			if !subscriptionStartDate.IsZero() && !subscriptionEndDate.IsZero() {
				if (drawStartDate.After(subscriptionStartDate) || drawStartDate.Equal(subscriptionStartDate)) &&
					(drawStartDate.Before(subscriptionEndDate) || drawStartDate.Equal(subscriptionEndDate)) {
					return e.JSON(http.StatusOK, map[string]interface{}{
						"hasAccess": true,
					})
				}
			}

			// Check if user has specifically paid for this draw
			filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, drawId)
			entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 1, 0)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Error checking draw access",
				})
			}

			if len(entries) > 0 {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"hasAccess": true,
				})
			}

			// If none of the access conditions were met
			return e.JSON(http.StatusOK, map[string]interface{}{
				"hasAccess": false,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
