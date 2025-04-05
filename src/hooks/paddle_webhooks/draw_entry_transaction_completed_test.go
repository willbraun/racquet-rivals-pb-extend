package paddle_webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func TestDrawEntryTransactionCompletedWebhook(t *testing.T) {
	// Load all mock data files for testing
	mockDataFiles := map[string]string{
		"Men":          filepath.Join("testdata", "mens_draw_entry.json"),
		"Women":        filepath.Join("testdata", "womens_draw_entry.json"),
		"Both":         filepath.Join("testdata", "both_draw_entry.json"),
		"Subscription": filepath.Join("testdata", "transaction_completed_subscription.json"),
	}

	mockDataStr := make(map[string]string)
	webhookData := make(map[string]PaddleDrawEntryTransaction)

	for productType, path := range mockDataFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read mock data for %s: %v", productType, err)
		}
		mockDataStr[productType] = string(data)

		var webhook PaddleDrawEntryTransaction
		if err := json.Unmarshal(data, &webhook); err != nil {
			t.Fatalf("Failed to parse mock data for %s: %v", productType, err)
		}
		webhookData[productType] = webhook
	}

	// Extract user and draw IDs
	userId := webhookData["Men"].Data.CustomData.UserID
	mensDrawId := *webhookData["Men"].Data.CustomData.MensDrawID
	womensDrawId := *webhookData["Women"].Data.CustomData.WomensDrawID

	// Test apps
	setupTestAppWithExistingEntry := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterDrawEntryTransactionCompletedHook(testApp)

		userDrawEntry, err := testApp.FindCollectionByNameOrId("user_draw_entry")
		if err != nil {
			t.Fatalf("Failed to find user_draw_entry collection: %v", err)
		}

		record := core.NewRecord(userDrawEntry)
		record.Set("user_id", userId)
		record.Set("draw_id", mensDrawId)

		if err := testApp.Save(record); err != nil {
			t.Fatalf("Failed to save user_draw_entry record: %v", err)
		}

		return testApp
	}

	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterDrawEntryTransactionCompletedHook(testApp)

		return testApp
	}

	// Test functions
	checkBeforeNonExistent := func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 0, len(entries), "user_draw_entry record should not exist before test")
	}

	checkBeforeExists := func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 1, len(entries), "user_draw_entry record should exist before idempotent test")

		if len(entries) > 0 {
			entry := entries[0]
			assert.Equal(t, userId, entry.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, mensDrawId, entry.GetString("draw_id"), "draw_id should match webhook data")
		}
	}

	checkMensEntryDoesNotExist := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 0, len(entries), "mens's user_draw_entry record should not exist after test")
	}

	checkMensEntryExists := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 1, len(entries), "mens's user_draw_entry record should exist after test")

		if len(entries) > 0 {
			entry := entries[0]
			assert.Equal(t, userId, entry.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, mensDrawId, entry.GetString("draw_id"), "draw_id should match webhook data")
		}
	}

	checkWomensEntryExists := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, womensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 1, len(entries), "women's user_draw_entry record should exist after test")

		if len(entries) > 0 {
			entry := entries[0]
			assert.Equal(t, userId, entry.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, womensDrawId, entry.GetString("draw_id"), "draw_id should match webhook data")
		}
	}

	checkBothEntriesExist := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		// Check men's entry
		mensFilter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		mensEntries, err := app.FindRecordsByFilter("user_draw_entry", mensFilter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing men's entries: %v", err)
		}
		assert.Equal(t, 1, len(mensEntries), "men's user_draw_entry record should exist after adding both draws")

		// Check women's entry
		womensFilter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, womensDrawId)
		womensEntries, err := app.FindRecordsByFilter("user_draw_entry", womensFilter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing women's entries: %v", err)
		}
		assert.Equal(t, 1, len(womensEntries), "women's user_draw_entry record should exist after adding both draws")
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "Successfully process men's transaction completed webhook",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr["Men"]),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry added successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkMensEntryExists,
		},
		{
			Name:           "Successfully process women's transaction completed webhook",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr["Women"]),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry added successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkWomensEntryExists,
		},
		{
			Name:           "Successfully process both draws transaction completed webhook",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr["Both"]),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry added successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkBothEntriesExist,
		},
		{
			Name:           "Idempotent request (sending the same webhook twice)",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr["Men"]),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry already exists, no new entry created"`,
			},
			TestAppFactory: setupTestAppWithExistingEntry,
			BeforeTestFunc: checkBeforeExists,
			AfterTestFunc:  checkMensEntryExists,
		},
		{
			Name:           "Transaction completed for subscription should succeed without creating draw entry",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr["Subscription"]),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"No action taken, subscription activation will be handled by the subscription.activated webhook."`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkMensEntryDoesNotExist,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestDrawEntryTransactionCompletedValidation tests various error conditions of the webhook
func TestDrawEntryTransactionCompletedValidation(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		RegisterDrawEntryTransactionCompletedHook(testApp)

		return testApp
	}

	// Load base mock data for modifications
	mockDataPath := filepath.Join("testdata", "mens_draw_entry.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var baseWebhook PaddleDrawEntryTransaction
	if err := json.Unmarshal(mockData, &baseWebhook); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	mensDrawID := "test-mens-draw-id"
	womensDrawID := "test-womens-draw-id"

	// Create modified test data based on test cases
	testCases := []struct {
		name        string
		modifyFunc  func(*PaddleDrawEntryTransaction)
		expectedMsg string
	}{
		{
			name: "Missing men's draw ID for men's product",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items[0].Price.ProductID = productIDs["Men"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = nil
			},
			expectedMsg: "men's draw entry must also have mens_draw_id in custom_data",
		},
		{
			name: "Missing women's draw ID for women's product",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items[0].Price.ProductID = productIDs["Women"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = nil
			},
			expectedMsg: "women's draw entry must also have womens_draw_id in custom_data",
		},
		{
			name: "Missing mens_draw_id for both draws product",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items[0].Price.ProductID = productIDs["Both"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			expectedMsg: "men's and women's joint entry must also have mens_draw_id and womens_draw_id in custom_data",
		},
		{
			name: "Missing womens_draw_id for both draws product",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items[0].Price.ProductID = productIDs["Both"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = nil
			},
			expectedMsg: "men's and women's joint entry must also have mens_draw_id and womens_draw_id in custom_data",
		},
		{
			name: "Missing event_id",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.EventID = ""
			},
			expectedMsg: "missing event_id",
		},
		{
			name: "Invalid event_type",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.EventType = "invalid.event"
			},
			expectedMsg: "invalid event_type. Expected transaction.completed, got invalid.event",
		},
		{
			name: "Missing user_id",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.CustomData.UserID = ""
			},
			expectedMsg: "missing user_id in custom_data",
		},
		{
			name: "No items in transaction",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items = []TransactionItem{}
			},
			expectedMsg: "no items in transaction",
		},
		{
			name: "Invalid product ID",
			modifyFunc: func(p *PaddleDrawEntryTransaction) {
				p.Data.Items[0].Price.ProductID = "invalid_product_id"
			},
			expectedMsg: "no valid products found",
		},
	}

	// Create API scenarios for each test case
	var scenarios []tests.ApiScenario

	// Add the basic cases
	scenarios = append(scenarios, []tests.ApiScenario{
		{
			Name:           "Bad request - empty body",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
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
			URL:            "/webhook/draw-entry-transaction-completed",
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
		testWebhook := PaddleDrawEntryTransaction{}
		if err := deepCopy(baseWebhook, &testWebhook); err != nil {
			t.Fatalf("Failed to create deep copy: %v", err)
		}

		tc.modifyFunc(&testWebhook)
		testWebhookStr, _ := json.Marshal(testWebhook)

		scenario := tests.ApiScenario{
			Name:           tc.name,
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
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

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func deepCopy(original any, copy any) error {
	data, err := json.Marshal(original)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, copy)
}
