package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
		return nil
	})

	// when a match result comes in, award points for correct predictions
	app.OnRecordBeforeUpdateRequest("draw_slot").Add(func(e *core.RecordUpdateEvent) error {
		name := e.Record.GetString("name")
		round := float64(e.Record.GetInt("round"))
		filter := fmt.Sprintf(`draw_slot_id="%s"`, e.Record.Id)

		view_predictions, err := app.Dao().FindRecordsByFilter("view_predictions", filter, "", -1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(view_predictions) == 0 {
			return nil
		}

		for _, vp := range view_predictions {
			record, err := app.Dao().FindRecordById("prediction", vp.GetString("id"))
			if err != nil {
				log.Panicln(err)
			}

			if strings.Contains(record.GetString("name"), name) {
				size := float64(vp.GetInt("size"))
				r16Round := math.Log2(size) - float64(3)
				
				points := 0
				switch round - r16Round {
				case 1:
					// Quarterfinal
					points = 1
				case 2:
					// Semifinal
					points = 2
				case 3:
					// Final
					points = 4
				case 4:
					// Winner
					points = 8
				}

				record.Set("points", points)
				if err := app.Dao().SaveRecord(record); err != nil {
    			return err
				}
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
