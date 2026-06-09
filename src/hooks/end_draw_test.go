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
// The draw already has end_date set in the test data, so each scenario resets it
// to a known sentinel date to distinguish "unchanged" from "updated by hook"
// script_user is required to make updates to the draw slots

// sentinelEndDate is a known past date used to detect when end_date is updated by the hook
var sentinelEndDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func TestEndDrawUpdate(t *testing.T) {
	assert := assert.New(t)

	const r16Slot15Id = "4pbsipqoncnd14h"
	const finalSlotId = "t1arj8fwjbcfltm"

	recordToken, err := generateRecordToken("user", "script_user")
	if err != nil {
		t.Fatal(err)
	}

	requestHeaders := map[string]string{
		"Authorization": recordToken,
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "Fill a non-final slot, end_date is not set",
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

				// Reset end_date to sentinel since test data has it pre-set
				draw, err := testApp.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}
				draw.Set("end_date", sentinelEndDate)
				if err := testApp.Save(draw); err != nil {
					log.Println("Error resetting end_date", err)
				}

				RegisterAllHooks(testApp)

				return testApp
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				emptySlots, err := app.FindRecordsByFilter("draw_slot", fmt.Sprintf(`draw_id="%s"&&name=""`, drawId), "", -1, 0)
				if err != nil {
					log.Println("Error accessing empty slots", err)
				}

				assert.Greater(len(emptySlots), 1)
				assert.Equal(sentinelEndDate, draw.GetDateTime("end_date").Time())
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				emptySlots, err := app.FindRecordsByFilter("draw_slot", fmt.Sprintf(`draw_id="%s"&&name=""`, drawId), "", -1, 0)
				if err != nil {
					log.Println("Error accessing empty slots", err)
				}

				assert.Greater(len(emptySlots), 1)
				assert.Equal(sentinelEndDate, draw.GetDateTime("end_date").Time())
			},
		},
		{
			Name:   "Fill the last empty slot, end_date is set",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", finalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    7,
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

				// Reset end_date to sentinel and fill all empty slots except the final one
				// BEFORE registering hooks so that intermediate saves don't trigger end_draw
				draw, err := testApp.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}
				draw.Set("end_date", sentinelEndDate)
				if err := testApp.Save(draw); err != nil {
					log.Println("Error resetting end_date", err)
				}

				emptySlotsBeforeWinner, err := testApp.FindRecordsByFilter("draw_slot", fmt.Sprintf(`draw_id="%s"&&name=""&&id!="%s"`, drawId, finalSlotId), "", -1, 0)
				if err != nil {
					log.Println("Error accessing empty slots", err)
				}

				for _, slot := range emptySlotsBeforeWinner {
					slot.Set("name", "Player")
					if err := testApp.Save(slot); err != nil {
						log.Println("Error saving slot", err)
					}
				}

				RegisterAllHooks(testApp)

				return testApp
			},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw", err)
				}

				emptySlots, err := app.FindRecordsByFilter("draw_slot", fmt.Sprintf(`draw_id="%s"&&name=""`, drawId), "", -1, 0)
				if err != nil {
					log.Println("Error accessing empty slots", err)
				}

				assert.Equal(1, len(emptySlots))
				assert.Equal(sentinelEndDate, draw.GetDateTime("end_date").Time())
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				draw, err := app.FindRecordById("draw", drawId)
				if err != nil {
					log.Println("Error accessing test draw,", err)
				}

				emptySlots, err := app.FindRecordsByFilter("draw_slot", fmt.Sprintf(`draw_id="%s"&&name=""`, drawId), "", -1, 0)
				if err != nil {
					log.Println("Error accessing empty slots", err)
				}

				assert.Equal(0, len(emptySlots))
				assert.NotEqual(sentinelEndDate, draw.GetDateTime("end_date").Time())
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
