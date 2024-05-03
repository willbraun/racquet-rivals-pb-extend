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

const drawId = "2l1hqqi8puodmjq"

func TestPredictionCloseUpdate(t *testing.T) {
	assert := assert.New(t)

	const r16Slot15Id = "4pbsipqoncnd14h"
	const r16Slot16Id = "7wg2gmjqutu1bky"
	const qfSlot1Id = "y6aj4a1vxfibcfv"
	filter := fmt.Sprintf(`draw_id="%s"&&round="%d"&&name!=""`, drawId, 3)

	recordToken, err := generateRecordToken("user", "script_user")
	if err != nil {
		t.Fatal(err)
	}

	requestHeaders := map[string]string{
		"Authorization": recordToken,
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
			RequestHeaders: requestHeaders,
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
			RequestHeaders: requestHeaders,
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
			RequestHeaders: requestHeaders,
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

func TestPointUpdate(t *testing.T) {
	setupTestApp := func(t *testing.T) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		bindAppHooks(testApp)

		return testApp
	}

	setupBeforeTest := func(assert *assert.Assertions, winningPrediction string, losingPrediction string) func(t *testing.T, app *tests.TestApp, e *echo.Echo) {
		return func(t *testing.T, app *tests.TestApp, e *echo.Echo) {
			win, err := app.Dao().FindRecordById("prediction", winningPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			lose, err := app.Dao().FindRecordById("prediction", losingPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			assert.Equal(win.GetInt("points"), 0)
			assert.Equal(lose.GetInt("points"), 0)
		}
	}

	setupAfterTest := func(assert *assert.Assertions, winningPrediction string, losingPrediction string, expectedPoints int) func(t *testing.T, app *tests.TestApp, res *http.Response) {
		return func(t *testing.T, app *tests.TestApp, res *http.Response) {
			win, err := app.Dao().FindRecordById("prediction", winningPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			lose, err := app.Dao().FindRecordById("prediction", losingPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			assert.Equal(win.GetInt("points"), expectedPoints)
			assert.Equal(lose.GetInt("points"), 0)
		}
	}

	assert := assert.New(t)

	const quarterfinalSlotId = "y6aj4a1vxfibcfv"
	const semifinalSlotId = "43objik3hkdl34k"
	const finalSlotId = "alsia0d829o6qox"
	const winnerSlotId = "t1arj8fwjbcfltm"

	const quarterfinalWinningPrediction = "ahsplh4qx7zkwkt"
	const semifinalWinningPrediction = "w0gko3g832lihmm"
	const finalWinningPrediction = "r9nhh355zrokdhi"
	const winnerWinningPrediction = "3x4hc8ikjugec9a"

	const quarterfinalLosingPrediction = "tagsi9hyse5i8rz"
	const semifinalLosingPrediction = "yfzbezk7pvkofn6"
	const finalLosingPrediction = "ohobwbtobq156mu"
	const winnerLosingPrediction = "edbd19e25ersljg"

	expectedEvents := map[string]int{
		"OnModelAfterUpdate":          2,
		"OnModelBeforeUpdate":         2,
		"OnRecordAfterUpdateRequest":  1,
		"OnRecordBeforeUpdateRequest": 1,
	}

	recordToken, err := generateRecordToken("user", "script_user")
	if err != nil {
		t.Fatal(err)
	}

	requestHeaders := map[string]string{
		"Authorization": recordToken,
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "Quarterfinal prediction result",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", quarterfinalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    4,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			RequestHeaders: requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents:  expectedEvents,
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, quarterfinalWinningPrediction, quarterfinalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, quarterfinalWinningPrediction, quarterfinalLosingPrediction, 1),
		},
		{
			Name:   "Semifinal prediction result",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", semifinalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    5,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			RequestHeaders: requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents:  expectedEvents,
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, semifinalWinningPrediction, semifinalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, semifinalWinningPrediction, semifinalLosingPrediction, 2),
		},
		{
			Name:   "Final prediction result",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", finalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    6,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			RequestHeaders: requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents:  expectedEvents,
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, finalWinningPrediction, finalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, finalWinningPrediction, finalLosingPrediction, 4),
		},
		{
			Name:   "Winner prediction result",
			Method: http.MethodPatch,
			Url:    fmt.Sprintf("/api/collections/draw_slot/records/%s", winnerSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    7,
				Position: 1,
				Name:     "Rybakina",
				Seed:     "(2)",
			}),
			RequestHeaders: requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			ExpectedEvents:  expectedEvents,
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, winnerWinningPrediction, winnerLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, winnerWinningPrediction, winnerLosingPrediction, 8),
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
