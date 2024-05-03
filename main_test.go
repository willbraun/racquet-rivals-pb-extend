package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tokens"
	"github.com/stretchr/testify/assert"
)

// The test data in test_pb_data has one draw of size 64
// The first two rounds are filled, and 14/16 are filled in the round of 16
// script_user is required to make updates to the draw slots

const testDataDir = "./test_pb_data"

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

	record, err := app.Dao().FindAuthRecordByUsername(collectionNameOrId, username)
	if err != nil {
		return "", err
	}

	return tokens.NewRecordAuthToken(app, record)
}

func TestPredictionCloseUpdate(t *testing.T) {
	assert := assert.New(t)

	const drawId = "2l1hqqi8puodmjq"
	const r16Slot15Id = "4pbsipqoncnd14h"
	const r16Slot16Id = "7wg2gmjqutu1bky"
	const qfSlot1Id = "y6aj4a1vxfibcfv"
	filter := fmt.Sprintf(`draw_id="%s"&&round="%d"&&name!=""`, drawId, 3)

	recordToken, err := generateRecordToken("user", "script_user")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "Add R16 slot 15, prediction close is not set",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", r16Slot15Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    3,
				Position: 15,
				Name:     "Mertens",
				Seed:     "(13)",
			}),
			RequestHeaders: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdate":          1,
				"OnModelBeforeUpdate":         1,
				"OnRecordAfterUpdateRequest":  1,
				"OnRecordBeforeUpdateRequest": 1,
			},
			TestAppFactory: func(t *testing.T) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				bindAppHooks(testApp)

				return testApp
			},
			BeforeTestFunc: func(t *testing.T, app *tests.TestApp, e *echo.Echo) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 14)
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t *testing.T, app *tests.TestApp, res *http.Response) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 15)
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
		},
		{
			Name:   "Add R16 slot 16, prediction close is set",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", r16Slot16Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    3,
				Position: 16,
				Name:     "Rybakina",
				Seed:     "(2)",
			}),
			RequestHeaders: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdate":          3,
				"OnModelBeforeUpdate":         3,
				"OnRecordAfterUpdateRequest":  1,
				"OnRecordBeforeUpdateRequest": 1,
			},
			TestAppFactory: func(t *testing.T) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				bindAppHooks(testApp)

				slot15, err := testApp.Dao().FindRecordById("draw_slot", r16Slot15Id)
				if err != nil {
					log.Println("Error accessing slot 15", err)
				}

				slot15.Set("name", "Mertens")
				if err := testApp.Dao().SaveRecord(slot15); err != nil {
					log.Println("Error saving slot 15", err)
				}

				return testApp
			},
			BeforeTestFunc: func(t *testing.T, app *tests.TestApp, e *echo.Echo) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 15)
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t *testing.T, app *tests.TestApp, res *http.Response) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 16)
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
		},
		{
			Name:   "Add QF slot 1, prediction close is still set",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", qfSlot1Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    4,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			RequestHeaders: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdate":          5,
				"OnModelBeforeUpdate":         5,
				"OnRecordAfterUpdateRequest":  1,
				"OnRecordBeforeUpdateRequest": 1,
			},
			TestAppFactory: func(t *testing.T) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				bindAppHooks(testApp)

				slot15, err := testApp.Dao().FindRecordById("draw_slot", r16Slot15Id)
				if err != nil {
					log.Println("Error accessing slot 15", err)
				}

				slot15.Set("name", "Mertens")
				if err := testApp.Dao().SaveRecord(slot15); err != nil {
					log.Println("Error saving slot 15", err)
				}

				slot16, err := testApp.Dao().FindRecordById("draw_slot", r16Slot16Id)
				if err != nil {
					log.Println("Error accessing slot 16", err)
				}

				slot16.Set("name", "Rybakina")
				if err := testApp.Dao().SaveRecord(slot16); err != nil {
					log.Println("Error saving slot 16", err)
				}

				draw, err := testApp.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				draw.Set("prediction_close", time.Now())
				if err := testApp.Dao().SaveRecord(draw); err != nil {
					log.Println("Error saving draw", err)
				}

				return testApp
			},
			BeforeTestFunc: func(t *testing.T, app *tests.TestApp, e *echo.Echo) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 16)
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t *testing.T, app *tests.TestApp, res *http.Response) {
				draw, err := app.Dao().FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(len(r16FilledSlots), 16)
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
