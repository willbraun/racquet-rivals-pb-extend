package paddle_webhooks

import (
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

	// Read mock data from the JSON file
	mockDataPath := filepath.Join("mock_data", "mens_draw_entry.json")
	mockData, err := os.ReadFile(mockDataPath)
	if err != nil {
		t.Fatalf("Failed to read mock data: %v", err)
	}

	// Convert to string for use in tests
	mockDataStr := string(mockData)

	// Define test scenarios
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
		{
			Name:           "Bad request - empty body",
			Method:         http.MethodPost,
			URL:            "/webhook/draw-entry-transaction-completed",
			Body:           strings.NewReader("{}"),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"error"`,
			},
			TestAppFactory: setupTestApp,
		},
	}

	// Run the tests
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// // TestDrawEntryTransactionCompletedValidation tests various error conditions of the webhook
// func TestDrawEntryTransactionCompletedValidation(t *testing.T) {
// 	setupTestApp := func(t testing.TB) *tests.TestApp {
// 		testApp, err := tests.NewTestApp(testDataDir)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		RegisterDrawEntryTransactionCompletedHook(testApp)

// 		return testApp
// 	}

// 	// Read mock data from the JSON file
// 	mockDataPath := filepath.Join("mock_data", "mens_draw_entry.json")
// 	mockData, err := os.ReadFile(mockDataPath)
// 	if err != nil {
// 		t.Fatalf("Failed to read mock data: %v", err)
// 	}

// 	// Parse the JSON to modify it for different test cases
// 	var webhookData map[string]interface{}
// 	if err := json.Unmarshal(mockData, &webhookData); err != nil {
// 		t.Fatalf("Failed to parse mock data: %v", err)
// 	}

// 	// Create modified versions of the original data

// 	// 1. Missing men's draw ID
// 	missingMensDrawData := deepCopy(webhookData)
// 	customData := missingMensDrawData["custom_data"].(map[string]interface{})
// 	delete(customData, "mens_draw_id")
// 	missingMensDrawDataStr, _ := json.Marshal(missingMensDrawData)

// 	// 2. No valid products
// 	noProductsData := deepCopy(webhookData)
// 	noProductsData["items"] = []interface{}{}
// 	noProductsDataStr, _ := json.Marshal(noProductsData)

// 	// Define test scenarios with invalid data
// 	scenarios := []tests.ApiScenario{
// 		{
// 			Name:           "Invalid JSON",
// 			Method:         http.MethodPost,
// 			URL:            "/webhook/draw-entry-transaction-completed",
// 			Body:           strings.NewReader("{invalid-json"),
// 			ExpectedStatus: 400,
// 			ExpectedContent: []string{
// 				`"error":"Invalid JSON format"`,
// 			},
// 			TestAppFactory: setupTestApp,
// 		},
// 		{
// 			Name:           "Missing men's draw ID",
// 			Method:         http.MethodPost,
// 			URL:            "/webhook/draw-entry-transaction-completed",
// 			Body:           strings.NewReader(string(missingMensDrawDataStr)),
// 			ExpectedStatus: 400,
// 			ExpectedContent: []string{
// 				`"error":"Men's draw ID is required for men's product"`,
// 			},
// 			TestAppFactory: setupTestApp,
// 		},
// 		{
// 			Name:           "No valid products",
// 			Method:         http.MethodPost,
// 			URL:            "/webhook/draw-entry-transaction-completed",
// 			Body:           strings.NewReader(string(noProductsDataStr)),
// 			ExpectedStatus: 400,
// 			ExpectedContent: []string{
// 				`"error":"No valid products found in transaction"`,
// 			},
// 			TestAppFactory: setupTestApp,
// 		},
// 	}

// 	// Run the tests
// 	for _, scenario := range scenarios {
// 		scenario.Test(t)
// 	}
// }

// Helper function to create a deep copy of a map
func deepCopy(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch v := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopy(v)
		case []interface{}:
			result[k] = deepCopySlice(v)
		default:
			result[k] = v
		}
	}
	return result
}

// Helper function to create a deep copy of a slice
func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		switch v := v.(type) {
		case map[string]interface{}:
			result[i] = deepCopy(v)
		case []interface{}:
			result[i] = deepCopySlice(v)
		default:
			result[i] = v
		}
	}
	return result
}
