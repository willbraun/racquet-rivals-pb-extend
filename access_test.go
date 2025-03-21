package main

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

		bindAppHooks(testApp)

		return testApp
	}

	const testDrawId = "2l1hqqi8puodmjq"

	t.Run("Access granted when user is grandfathered", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		defer app.Cleanup()

		bindAppHooks(app)

		grandfatheredUserId := "r8xmngx0sgau633"

		scenario := tests.ApiScenario{
			Name:            "Grandfathered user access",
			Method:          http.MethodGet,
			URL:             "/access/" + grandfatheredUserId + "/" + testDrawId,
			ExpectedStatus:  200,
			ExpectedContent: []string{`"hasAccess":true`},
			TestAppFactory:  setupTestApp,
		}

		scenario.Test(t)
	})

	t.Run("Access granted when user has active subscription", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		defer app.Cleanup()

		bindAppHooks(app)

		subscribedUserId := "r8xmngx0sgau633"

		scenario := tests.ApiScenario{
			Name:            "Subscription user access",
			Method:          http.MethodGet,
			URL:             "/access/" + subscribedUserId + "/" + testDrawId,
			ExpectedStatus:  200,
			ExpectedContent: []string{`"hasAccess":true`},
			TestAppFactory:  setupTestApp,
		}

		scenario.Test(t)
	})

	t.Run("Access granted when user has draw entry", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		defer app.Cleanup()

		bindAppHooks(app)

		userWithDrawEntryId := "e425j00gwm5pi1h"

		scenario := tests.ApiScenario{
			Name:            "Draw entry user access",
			Method:          http.MethodGet,
			URL:             "/access/" + userWithDrawEntryId + "/" + testDrawId,
			ExpectedStatus:  200,
			ExpectedContent: []string{`"hasAccess":true`},
			TestAppFactory:  setupTestApp,
		}

		scenario.Test(t)
	})

	t.Run("Access denied when user meets no criteria", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		defer app.Cleanup()

		bindAppHooks(app)

		userWithNoAccessId := "2y5ysc47g9l6ebq"

		scenario := tests.ApiScenario{
			Name:            "No access user",
			Method:          http.MethodGet,
			URL:             "/access/" + userWithNoAccessId + "/" + testDrawId,
			ExpectedStatus:  200,
			ExpectedContent: []string{`"hasAccess":false`},
			TestAppFactory:  setupTestApp,
		}

		scenario.Test(t)
	})
}
