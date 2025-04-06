package paddle_webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func TestSubscriptionActivatedWebhook(t *testing.T) {
	// Load all mock data files for testing
	mockDataPath := filepath.Join("testdata", "subscription_activated.json")

	data, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var mockPayload PaddleSubscriptionActivated
	if err := json.Unmarshal(data, &mockPayload); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}
	mockPayloadStr := string(data)

	// Extract user and subscription IDs
	userId := mockPayload.Data.CustomData.UserID
	subscriptionId := mockPayload.Data.ID

	// Test apps
	setupTestAppWithExistingSubscription := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionActivatedHook(testApp)

		subscription, err := testApp.FindCollectionByNameOrId("subscription")
		if err != nil {
			t.Fatalf("Failed to find subscription collection: %v", err)
		}

		record := core.NewRecord(subscription)
		record.Set("user_id", userId)
		record.Set("paddle_subscription_id", subscriptionId)
		record.Set("status", "active")
		record.Set("current_billing_period_start", time.Now())
		record.Set("current_billing_period_end", time.Now().Add(30*24*time.Hour))

		if err := testApp.Save(record); err != nil {
			t.Fatalf("Failed to save subscription record: %v", err)
		}

		return testApp
	}

	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionActivatedHook(testApp)

		return testApp
	}

	// Test functions
	checkBeforeNonExistent := func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		filter := fmt.Sprintf(`user_id="%s"`, userId)
		subscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing subscriptions: %v", err)
		}

		assert.Equal(t, 0, len(subscriptions), "subscription record should not exist before test")
	}

	checkBeforeExists := func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		filter := fmt.Sprintf(`user_id="%s"`, userId)
		subscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing subscriptions: %v", err)
		}

		assert.Equal(t, 1, len(subscriptions), "subscription record should exist before idempotent test")

		if len(subscriptions) > 0 {
			subscription := subscriptions[0]
			assert.Equal(t, userId, subscription.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, subscriptionId, subscription.GetString("paddle_subscription_id"), "paddle_subscription_id should match webhook data")
			assert.Equal(t, "active", subscription.GetString("status"), "status should be active")
		}
	}

	checkSubscriptionExists := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`user_id="%s"`, userId)
		subscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing subscriptions: %v", err)
		}

		assert.Equal(t, 1, len(subscriptions), "subscription record should exist after test")

		if len(subscriptions) > 0 {
			subscription := subscriptions[0]
			assert.Equal(t, userId, subscription.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, subscriptionId, subscription.GetString("paddle_subscription_id"), "paddle_subscription_id should match webhook data")
			assert.Equal(t, "active", subscription.GetString("status"), "status should be active")
			assert.False(t, subscription.GetDateTime("current_billing_period_start").IsZero(), "current_billing_period_start should not be zero")
			assert.False(t, subscription.GetDateTime("current_billing_period_end").IsZero(), "current_billing_period_end should not be zero")
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "Successfully process subscription activated webhook",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-activated",
			Body:           strings.NewReader(mockPayloadStr),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Subscription activated successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkSubscriptionExists,
		},
		{
			Name:           "Idempotent request (sending the same webhook twice)",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-activated",
			Body:           strings.NewReader(mockPayloadStr),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"User already has an active subscription, no new subscription created"`,
			},
			TestAppFactory: setupTestAppWithExistingSubscription,
			BeforeTestFunc: checkBeforeExists,
			AfterTestFunc:  checkSubscriptionExists,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestSubscriptionActivatedValidation tests various error conditions of the webhook
func TestSubscriptionActivatedValidation(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionActivatedHook(testApp)

		return testApp
	}

	// Load base mock data for modifications
	mockDataPath := filepath.Join("testdata", "subscription_activated.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var baseWebhook PaddleSubscriptionActivated
	if err := json.Unmarshal(mockData, &baseWebhook); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	// Create modified test data based on test cases
	testCases := []struct {
		name        string
		modifyFunc  func(*PaddleSubscriptionActivated)
		expectedMsg string
	}{
		{
			name: "Missing event_id",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.EventID = ""
			},
			expectedMsg: "missing event_id",
		},
		{
			name: "Invalid event_type",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.EventType = "invalid.type"
			},
			expectedMsg: "invalid event_type. Expected subscription.activated, got invalid.type",
		},
		{
			name: "Missing user_id",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.CustomData.UserID = ""
			},
			expectedMsg: "missing user_id in custom_data",
		},
		{
			name: "No items in subscription",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.Items = []SubscriptionItem{}
			},
			expectedMsg: "no items in transaction",
		},
		{
			name: "Inactive subscription status",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.Status = "paused"
			},
			expectedMsg: "subscription status is not active: paused",
		},
		{
			name: "Incorrect Product ID",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.Items[0].Product.ID = "invalid_product_id"
			},
			expectedMsg: "invalid product ID. Expected pro_01jpkhsd61k1acva107vz6dj0v, got invalid_product_id",
		},
		{
			name: "Missing billing period start date",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.CurrentBillingPeriod.StartsAt = time.Time{}
			},
			expectedMsg: "missing billing period dates",
		},
		{
			name: "Missing billing period end date",
			modifyFunc: func(p *PaddleSubscriptionActivated) {
				p.Data.CurrentBillingPeriod.EndsAt = time.Time{}
			},
			expectedMsg: "missing billing period dates",
		},
	}

	// Create API scenarios for each test case
	var scenarios []tests.ApiScenario

	// Add the basic cases
	scenarios = append(scenarios, []tests.ApiScenario{
		{
			Name:           "Bad request - empty body",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-activated",
			Body:           strings.NewReader("{}"),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"error":"Invalid webhook payload format"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:           "Invalid JSON",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-activated",
			Body:           strings.NewReader("{invalid-json"),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"error":"Invalid JSON format"`,
			},
			TestAppFactory: setupTestApp,
		},
	}...)

	// Then add test cases for each validation case
	for _, tc := range testCases {
		testWebhook := PaddleSubscriptionActivated{}
		if err := deepCopy(baseWebhook, &testWebhook); err != nil {
			t.Fatalf("Failed to create deep copy: %v", err)
		}

		tc.modifyFunc(&testWebhook)
		testWebhookStr, _ := json.Marshal(testWebhook)

		scenario := tests.ApiScenario{
			Name:           tc.name,
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-activated",
			Body:           strings.NewReader(string(testWebhookStr)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				fmt.Sprintf(`"details":"%s"`, tc.expectedMsg),
				`"error":"Invalid webhook payload format"`,
			},
			TestAppFactory: setupTestApp,
		}

		scenarios = append(scenarios, scenario)
	}

	// Missing user test case (this returns 404 instead of 400)
	badUserWebhook := PaddleSubscriptionActivated{}
	if err := deepCopy(baseWebhook, &badUserWebhook); err != nil {
		t.Fatalf("Failed to create deep copy: %v", err)
	}
	badUserWebhook.Data.CustomData.UserID = "nonexistent_user_id"
	badUserWebhookStr, _ := json.Marshal(badUserWebhook)

	scenarios = append(scenarios, tests.ApiScenario{
		Name:           "User not found",
		Method:         http.MethodPost,
		URL:            "/webhook/subscription-activated",
		Body:           strings.NewReader(string(badUserWebhookStr)),
		ExpectedStatus: 404,
		ExpectedContent: []string{
			`"error":"User with ID 'nonexistent_user_id' not found"`,
		},
		TestAppFactory: setupTestApp,
	})

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
