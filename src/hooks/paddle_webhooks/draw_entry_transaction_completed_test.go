package paddle_webhooks

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

const testDataDir = "../../../test_pb_data"

func TestDrawEntryTransactionCompletedWebhook(t *testing.T) {
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

	mockDataStr := string(mockData)

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
			TestAppFactory: setupTestApp,
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

func deepCopy(original any, copy any) error {
	data, err := json.Marshal(original)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, copy)
}
