package paddle_webhooks

import (
	"database/sql"
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

func TestSubscriptionUpdatedWebhook(t *testing.T) {
	// Load all mock data files for testing
	mockDataPath := filepath.Join("testdata", "subscription_updated.json")

	data, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var mockPayload PaddleSubscriptionUpdated
	if err := json.Unmarshal(data, &mockPayload); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	// Update the subscription ID to match the test data
	mockPayload.Data.ID = "sub_example_paddle_subscription_id"
	mockPayloadBytes, _ := json.Marshal(mockPayload)
	mockPayloadStr := string(mockPayloadBytes)

	// Extract subscription ID
	subscriptionId := mockPayload.Data.ID

	// Test app setup
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionUpdatedHook(testApp)

		return testApp
	}

	// Test functions
	checkBeforeExists := func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		filter := fmt.Sprintf(`paddle_subscription_id="%s"`, subscriptionId)
		subscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing subscriptions: %v", err)
		}

		assert.Equal(t, 1, len(subscriptions), "subscription record should exist before test")

		if len(subscriptions) > 0 {
			subscription := subscriptions[0]
			assert.Equal(t, subscriptionId, subscription.GetString("paddle_subscription_id"), "paddle_subscription_id should match webhook data")
			assert.Equal(t, "active", subscription.GetString("status"), "status should be active initially")
		}
	}

	checkSubscriptionUpdated := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`paddle_subscription_id="%s"`, subscriptionId)
		subscriptions, err := app.FindRecordsByFilter("subscription", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing subscriptions: %v", err)
		}

		assert.Equal(t, 1, len(subscriptions), "subscription record should exist after test")

		if len(subscriptions) > 0 {
			subscription := subscriptions[0]
			assert.Equal(t, subscriptionId, subscription.GetString("paddle_subscription_id"), "paddle_subscription_id should remain unchanged")
			assert.Equal(t, "canceled", subscription.GetString("status"), "status should be updated to canceled")

			// Verify billing period dates were updated
			startDate := subscription.GetDateTime("current_billing_period_start")
			endDate := subscription.GetDateTime("current_billing_period_end")

			expectedStart, _ := time.Parse(time.RFC3339, "2026-04-12T10:18:47.635628Z")
			expectedEnd, _ := time.Parse(time.RFC3339, "2026-05-12T10:18:47.635628Z")

			// Convert types.DateTime to time.Time and truncate to seconds for comparison
			startTime := startDate.Time().Truncate(time.Second)
			endTime := endDate.Time().Truncate(time.Second)
			expectedStartTruncated := expectedStart.Truncate(time.Second)
			expectedEndTruncated := expectedEnd.Truncate(time.Second)

			assert.Equal(t, expectedStartTruncated, startTime, "current_billing_period_start should match webhook data (to the nearest second)")
			assert.Equal(t, expectedEndTruncated, endTime, "current_billing_period_end should match webhook data (to the nearest second)")
		}
	}

	// Create a test app for missing subscription
	setupTestAppWithoutSubscription := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionUpdatedHook(testApp)

		// Delete the subscription record if it exists
		filter := fmt.Sprintf(`paddle_subscription_id="%s"`, subscriptionId)
		record, err := testApp.FindFirstRecordByFilter("subscription", filter)
		if err != nil {
			if err != sql.ErrNoRows {
				t.Fatalf("Failed to find subscription record: %v", err)
			}
		}

		if err := testApp.Delete(record); err != nil {
			t.Fatalf("Failed to delete existing subscription record: %v", err)
		}

		return testApp
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "Successfully update subscription",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-updated",
			Body:           strings.NewReader(mockPayloadStr),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Subscription updated successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeExists,
			AfterTestFunc:  checkSubscriptionUpdated,
		},
		{
			Name:           "Subscription not found",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-updated",
			Body:           strings.NewReader(mockPayloadStr),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`{"data":{},"message":"Subscription with paddle_subscription_id 'sub_example_paddle_subscription_id' not found.","status":404}`,
			},
			TestAppFactory: setupTestAppWithoutSubscription,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestSubscriptionUpdatedValidation tests various error conditions of the webhook
func TestSubscriptionUpdatedValidation(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterSubscriptionUpdatedHook(testApp)

		return testApp
	}

	// Load base mock data for modifications
	mockDataPath := filepath.Join("testdata", "subscription_updated.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var baseWebhook PaddleSubscriptionUpdated
	if err := json.Unmarshal(mockData, &baseWebhook); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	// Update the subscription ID to match the test data
	baseWebhook.Data.ID = "sub_example_paddle_subscription_id"

	// Create modified test data based on test cases
	testCases := []struct {
		name        string
		modifyFunc  func(*PaddleSubscriptionUpdated)
		expectedMsg string
	}{
		{
			name: "Missing event_id",
			modifyFunc: func(p *PaddleSubscriptionUpdated) {
				p.EventID = ""
			},
			expectedMsg: "missing event_id",
		},
		{
			name: "Invalid event_type",
			modifyFunc: func(p *PaddleSubscriptionUpdated) {
				p.EventType = "invalid.type"
			},
			expectedMsg: "invalid event_type. Expected subscription.updated, got invalid.type",
		},
		{
			name: "No items in subscription",
			modifyFunc: func(p *PaddleSubscriptionUpdated) {
				p.Data.Items = []SubscriptionItem{}
			},
			expectedMsg: "no items in transaction",
		},
		{
			name: "Incorrect Product ID",
			modifyFunc: func(p *PaddleSubscriptionUpdated) {
				p.Data.Items[0].Product.ID = "invalid_product_id"
			},
			expectedMsg: fmt.Sprintf("invalid product ID. Expected %s, got invalid_product_id", os.Getenv("SUBSCRIPTION_PRODUCT_ID")),
		},
	}

	// Create API scenarios for each test case
	var scenarios []tests.ApiScenario

	// Add the basic cases
	scenarios = append(scenarios, []tests.ApiScenario{
		{
			Name:           "Bad request - empty body",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-updated",
			Body:           strings.NewReader("{}"),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"data":{},"message":"Invalid webhook payload format: missing event_id.","status":400}`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:           "Invalid JSON",
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-updated",
			Body:           strings.NewReader("{invalid-json"),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"data":{},"message":"Invalid JSON format.","status":400}`,
			},
			TestAppFactory: setupTestApp,
		},
	}...)

	// Then add test cases for each validation case
	for _, tc := range testCases {
		testWebhook := PaddleSubscriptionUpdated{}
		if err := deepCopy(baseWebhook, &testWebhook); err != nil {
			t.Fatalf("Failed to create deep copy: %v", err)
		}

		tc.modifyFunc(&testWebhook)
		testWebhookStr, _ := json.Marshal(testWebhook)

		scenario := tests.ApiScenario{
			Name:           tc.name,
			Method:         http.MethodPost,
			URL:            "/webhook/subscription-updated",
			Body:           strings.NewReader(string(testWebhookStr)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				fmt.Sprintf(`{"data":{},"message":"Invalid webhook payload format: %s.","status":400}`, tc.expectedMsg),
			},
			TestAppFactory: setupTestApp,
		}

		scenarios = append(scenarios, scenario)
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
