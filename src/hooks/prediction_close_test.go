package hooks

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

// The test data in test_pb_data has one draw of size 64
// The first two rounds are filled, and 14/16 are filled in the round of 16
// script_user is required to make updates to the draw slots

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
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", r16Slot15Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    3,
				Position: 15,
				Name:     "Mertens",
				Seed:     "(13)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				RegisterAllHooks(testApp)

				return testApp
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(14, len(r16FilledSlots))
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(15, len(r16FilledSlots))
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
		},
		{
			Name:   "Add R16 slot 16, prediction close is set",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", r16Slot16Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    3,
				Position: 16,
				Name:     "Rybakina",
				Seed:     "(2)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				RegisterAllHooks(testApp)

				slot15, err := testApp.FindRecordById("draw_slot", r16Slot15Id)
				if err != nil {
					log.Println("Error accessing slot 15", err)
				}

				slot15.Set("name", "Mertens")
				if err := testApp.Save(slot15); err != nil {
					log.Println("Error saving slot 15", err)
				}

				return testApp
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(15, len(r16FilledSlots))
				assert.Empty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(16, len(r16FilledSlots))
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
		},
		{
			Name:   "Add QF slot 1, prediction close is still set",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", qfSlot1Id),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    4,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				testApp, err := tests.NewTestApp(testDataDir)
				if err != nil {
					t.Fatal(err)
				}

				RegisterAllHooks(testApp)

				slot15, err := testApp.FindRecordById("draw_slot", r16Slot15Id)
				if err != nil {
					log.Println("Error accessing slot 15", err)
				}

				slot15.Set("name", "Mertens")
				if err := testApp.Save(slot15); err != nil {
					log.Println("Error saving slot 15", err)
				}

				slot16, err := testApp.FindRecordById("draw_slot", r16Slot16Id)
				if err != nil {
					log.Println("Error accessing slot 16", err)
				}

				slot16.Set("name", "Rybakina")
				if err := testApp.Save(slot16); err != nil {
					log.Println("Error saving slot 16", err)
				}

				draw, err := testApp.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				draw.Set("prediction_close", time.Now())
				if err := testApp.Save(draw); err != nil {
					log.Println("Error saving draw", err)
				}

				return testApp
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(16, len(r16FilledSlots))
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
				if err != nil {
					log.Println("Error accessing round of 16 slots", err)
				}

				assert.Equal(16, len(r16FilledSlots))
				assert.NotEmpty(draw.GetDateTime("prediction_close"))
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
