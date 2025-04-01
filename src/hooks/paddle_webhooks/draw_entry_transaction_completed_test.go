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

const testDataDir = "../../../test_pb_data"

func TestDrawEntryTransactionCompletedWebhook(t *testing.T) {
	mockDataPath := filepath.Join("mock_data", "mens_draw_entry.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	mockDataStr := string(mockData)

	var webhookData PaddleTransactionCompleted
	if err := json.Unmarshal(mockData, &webhookData); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	userId := webhookData.Data.CustomData.UserID
	mensDrawId := *webhookData.Data.CustomData.MensDrawID

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

	checkAfterExists := func(t testing.TB, app *tests.TestApp, res *http.Response) {
		filter := fmt.Sprintf(`user_id="%s" && draw_id="%s"`, userId, mensDrawId)
		entries, err := app.FindRecordsByFilter("user_draw_entry", filter, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to check for existing entries: %v", err)
		}

		assert.Equal(t, 1, len(entries), "user_draw_entry record should exist after test")

		if len(entries) > 0 {
			entry := entries[0]
			assert.Equal(t, userId, entry.GetString("user_id"), "user_id should match webhook data")
			assert.Equal(t, mensDrawId, entry.GetString("draw_id"), "draw_id should match webhook data")
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "Successfully process transaction completed webhook",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry added successfully"`,
			},
			TestAppFactory: setupTestApp,
			BeforeTestFunc: checkBeforeNonExistent,
			AfterTestFunc:  checkAfterExists,
		},
		{
			Name:           "Idempotent request (sending the same webhook twice)",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(mockDataStr),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Draw entry added successfully"`,
			},
			TestAppFactory: setupTestAppWithExistingEntry,
			BeforeTestFunc: checkBeforeExists,
			AfterTestFunc:  checkAfterExists,
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

	mockDataPath := filepath.Join("mock_data", "mens_draw_entry.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var webhookData PaddleTransactionCompleted
	if err := json.Unmarshal(mockData, &webhookData); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	// Create modified versions of the original data

	// 1. Missing men's draw ID
	missingMensDrawData := PaddleTransactionCompleted{}
	if err := deepCopy(webhookData, &missingMensDrawData); err != nil {
		t.Fatalf("Failed to create deep copy: %v", err)
	}

	missingMensDrawData.Data.CustomData.MensDrawID = nil
	missingMensDrawDataStr, _ := json.Marshal(missingMensDrawData)

	// 2. No valid products
	noProductsData := PaddleTransactionCompleted{}
	if err := deepCopy(webhookData, &noProductsData); err != nil {
		t.Fatalf("Failed to create deep copy: %v", err)
	}

	noProductsData.Data.Items = []TransactionItem{}
	noProductsDataStr, _ := json.Marshal(noProductsData)

	scenarios := []tests.ApiScenario{
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
		{
			Name:           "Missing men's draw ID",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(string(missingMensDrawDataStr)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"details":"men's draw entry must also have mens_draw_id in custom_data","error":"Invalid webhook payload format"}`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:           "No valid products",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader(string(noProductsDataStr)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"details":"no items in transaction","error":"Invalid webhook payload format"}`,
			},
			TestAppFactory: setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestValidateWebhookPayload tests the validateWebhookPayload function directly
func TestValidateWebhookPayload(t *testing.T) {
	// Load the base test data to use as a template
	mockDataPath := filepath.Join("mock_data", "mens_draw_entry.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	var baseWebhook PaddleTransactionCompleted
	if err := json.Unmarshal(mockData, &baseWebhook); err != nil {
		t.Fatalf("Failed to parse mock data: %v", err)
	}

	mensDrawID := "test-mens-draw-id"
	womensDrawID := "test-womens-draw-id"

	// Create valid test cases for each product type
	tests := []struct {
		name    string
		modify  func(*PaddleTransactionCompleted)
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid men's draw",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Men"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = nil
			},
			wantErr: false,
		},
		{
			name: "Valid men's draw, with both draws present",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Men"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			wantErr: false,
		},
		{
			name: "Valid women's draw",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Women"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			wantErr: false,
		},
		{
			name: "Valid women's draw, with both draws present",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Women"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			wantErr: false,
		},
		{
			name: "Valid both draws",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Both"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			wantErr: false,
		},
		{
			name: "Missing men's draw ID for men's product",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Men"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = nil
			},
			wantErr: true,
			errMsg:  "men's draw entry must also have mens_draw_id in custom_data",
		},
		{
			name: "Missing women's draw ID for women's product",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Women"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = nil
			},
			wantErr: true,
			errMsg:  "women's draw entry must also have womens_draw_id in custom_data",
		},
		{
			name: "Missing mens_draw_id for both draws product",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Both"]
				p.Data.CustomData.MensDrawID = nil
				p.Data.CustomData.WomensDrawID = &womensDrawID
			},
			wantErr: true,
			errMsg:  "men's & women's joint entry must also have mens_draw_id and womens_draw_id in custom_data",
		},
		{
			name: "Missing womens_draw_id for both draws product",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = productIDs["Both"]
				p.Data.CustomData.MensDrawID = &mensDrawID
				p.Data.CustomData.WomensDrawID = nil
			},
			wantErr: true,
			errMsg:  "men's & women's joint entry must also have mens_draw_id and womens_draw_id in custom_data",
		},
		{
			name: "Missing event_id",
			modify: func(p *PaddleTransactionCompleted) {
				p.EventID = ""
			},
			wantErr: true,
			errMsg:  "missing event_id",
		},
		{
			name: "Invalid event_type",
			modify: func(p *PaddleTransactionCompleted) {
				p.EventType = "invalid.event"
			},
			wantErr: true,
			errMsg:  "invalid event_type",
		},
		{
			name: "Missing user_id",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.CustomData.UserID = ""
			},
			wantErr: true,
			errMsg:  "missing user_id in custom_data",
		},
		{
			name: "No items in transaction",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items = []TransactionItem{}
			},
			wantErr: true,
			errMsg:  "no items in transaction",
		},
		{
			name: "Invalid product ID",
			modify: func(p *PaddleTransactionCompleted) {
				p.Data.Items[0].Price.ProductID = "invalid_product_id"
			},
			wantErr: true,
			errMsg:  "no valid products found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a copy of the base webhook
			testWebhook := PaddleTransactionCompleted{}
			if err := deepCopy(baseWebhook, &testWebhook); err != nil {
				t.Fatalf("Failed to create deep copy: %v", err)
			}

			// Apply modifications specific to this test case
			test.modify(&testWebhook)

			// Call validateWebhookPayload
			err := validateWebhookPayload(testWebhook)

			// Check if error was expected
			if (err != nil) != test.wantErr {
				t.Errorf("validateWebhookPayload() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			// If error was expected, check if it contains the expected message
			if test.wantErr && err != nil && !strings.Contains(err.Error(), test.errMsg) {
				t.Errorf("validateWebhookPayload() error message = %v, expected to contain %v", err, test.errMsg)
			}
		})
	}
}

func deepCopy(original any, copy any) error {
	data, err := json.Marshal(original)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, copy)
}
