package tests

import (
	"fmt"
	"log"
	"net/http"
	"pocketbase_extend/src/hooks"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func TestAwardPoints(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		hooks.RegisterAllHooks(testApp)

		return testApp
	}

	setupBeforeTest := func(assert *assert.Assertions, winningPrediction string, losingPrediction string) func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		return func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			win, err := app.FindRecordById("prediction", winningPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			lose, err := app.FindRecordById("prediction", losingPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			assert.Equal(0, win.GetInt("points"))
			assert.Equal(0, lose.GetInt("points"))
		}
	}

	setupAfterTest := func(assert *assert.Assertions, winningPrediction string, losingPrediction string, expectedPoints int) func(t testing.TB, app *tests.TestApp, res *http.Response) {
		return func(t testing.TB, app *tests.TestApp, res *http.Response) {
			win, err := app.FindRecordById("prediction", winningPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			lose, err := app.FindRecordById("prediction", losingPrediction)
			if err != nil {
				log.Println("Error accessing winning prediction,", err)
			}

			assert.Equal(expectedPoints, win.GetInt("points"))
			assert.Equal(0, lose.GetInt("points"))
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

	recordToken, err := generateRecordToken("user", "script_user")
	if err != nil {
		t.Fatal(err)
	}

	requestHeaders := map[string]string{
		"Authorization": recordToken,
		"Content-Type":  "application/json",
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "Quarterfinal prediction result",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", quarterfinalSlotId),
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
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, quarterfinalWinningPrediction, quarterfinalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, quarterfinalWinningPrediction, quarterfinalLosingPrediction, 1),
		},
		{
			Name:   "Semifinal prediction result",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", semifinalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    5,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, semifinalWinningPrediction, semifinalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, semifinalWinningPrediction, semifinalLosingPrediction, 2),
		},
		{
			Name:   "Final prediction result",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", finalSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    6,
				Position: 1,
				Name:     "Sabalenka",
				Seed:     "(1)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, finalWinningPrediction, finalLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, finalWinningPrediction, finalLosingPrediction, 4),
		},
		{
			Name:   "Winner prediction result",
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/collections/draw_slot/records/%s", winnerSlotId),
			Body: getIoReaderBody(CreateUpdateSlotReq{
				DrawID:   drawId,
				Round:    7,
				Position: 1,
				Name:     "Rybakina",
				Seed:     "(2)",
			}),
			Headers:         requestHeaders,
			ExpectedStatus:  200,
			ExpectedContent: []string{"\"collectionName\":\"draw_slot\""},
			TestAppFactory:  setupTestApp,
			BeforeTestFunc:  setupBeforeTest(assert, winnerWinningPrediction, winnerLosingPrediction),
			AfterTestFunc:   setupAfterTest(assert, winnerWinningPrediction, winnerLosingPrediction, 8),
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
