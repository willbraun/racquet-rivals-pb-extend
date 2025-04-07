package hooks

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestAccessEndpoint(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterAllHooks(testApp)

		return testApp
	}

	const testDrawId = "2l1hqqi8puodmjq"
	const grandfatheredUserId = "rj8l4hni6tl4bas"
	const subscribedUserId = "r8xmngx0sgau633"
	const subscribedUserPastDueId = "un5y5qn29vt3711"
	const subscribedUserCanceledId = "y0p92fy9oj66ab1"
	const singleDrawUserId = "e425j00gwm5pi1h"
	const noAccessUserId = "2y5ysc47g9l6ebq"

	t.Run("Access is correct", func(t *testing.T) {
		scenarios := []tests.ApiScenario{
			{
				Name:            "Grandfathered user access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/" + grandfatheredUserId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":true`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Subscribed user with active subscription has access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("subscribed_user", t),
				URL:             "/access/" + subscribedUserId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":true`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Subscribed user with past due subscription has access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("subscribed_user_past_due", t),
				URL:             "/access/" + subscribedUserPastDueId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":true`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Subscribed user with canceled subscription does not have access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("subscribed_user_canceled", t),
				URL:             "/access/" + subscribedUserCanceledId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":false`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "User with draw entry access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("single_draw_user", t),
				URL:             "/access/" + singleDrawUserId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":true`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "User with no access",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("no_access_user", t),
				URL:             "/access/" + noAccessUserId + "/" + testDrawId,
				ExpectedStatus:  200,
				ExpectedContent: []string{`"hasAccess":false`},
				TestAppFactory:  setupTestApp,
			},
		}

		for _, scenario := range scenarios {
			scenario.Test(t)
		}
	})

	t.Run("Throws expected exceptions", func(t *testing.T) {
		scenarios := []tests.ApiScenario{
			{
				Name:            "Unauthorized",
				Method:          http.MethodGet,
				URL:             "/access/" + grandfatheredUserId + "/" + testDrawId,
				ExpectedStatus:  401,
				ExpectedContent: []string{`{"data":{},"message":"The request requires valid record authorization token.","status":401}`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Permission denied",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/" + subscribedUserId + "/" + testDrawId,
				ExpectedStatus:  403,
				ExpectedContent: []string{`{"data":{},"message":"You don't have permission to access subscribed_user's data.","status":403}`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "User not found",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/invalid_user_id/" + testDrawId,
				ExpectedStatus:  404,
				ExpectedContent: []string{`{"data":{},"message":"User with ID 'invalid_user_id' not found.","status":404}`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Draw not found",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/" + grandfatheredUserId + "/invalid_draw_id",
				ExpectedStatus:  404,
				ExpectedContent: []string{`{"data":{},"message":"Draw with ID 'invalid_draw_id' not found.","status":404}`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Missing user_id",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/%20/" + testDrawId,
				ExpectedStatus:  400,
				ExpectedContent: []string{`{"data":{},"message":"Must provide user_id.","status":400}`},
				TestAppFactory:  setupTestApp,
			},
			{
				Name:            "Missing draw_id",
				Method:          http.MethodGet,
				Headers:         createAuthHeader("grandfathered_user", t),
				URL:             "/access/" + grandfatheredUserId + "/%20",
				ExpectedStatus:  400,
				ExpectedContent: []string{`{"data":{},"message":"Must provide draw_id.","status":400}`},
				TestAppFactory:  setupTestApp,
			},
		}

		for _, scenario := range scenarios {
			scenario.Test(t)
		}
	})
}
