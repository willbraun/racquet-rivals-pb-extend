package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

const testDataDir = "./../../test_pb_data"
const drawId = "2l1hqqi8puodmjq"

type CreateUpdateSlotReq struct {
	DrawID   string `json:"draw_id"`
	Round    int    `json:"round"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	Seed     string `json:"seed"`
}

func getIoReaderBody(body CreateUpdateSlotReq) io.Reader {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Println("Error:", err)
		return strings.NewReader("")
	}

	return strings.NewReader(string(bodyBytes))
}

func generateRecordToken(collectionNameOrId string, username string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindFirstRecordByData(collectionNameOrId, "username", username)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}

func createAuthHeader(username string, t *testing.T) map[string]string {
	recordToken, err := generateRecordToken("user", username)
	if err != nil {
		t.Fatal(err)
	}

	requestHeaders := map[string]string{
		"Authorization": recordToken,
	}

	return requestHeaders
}
