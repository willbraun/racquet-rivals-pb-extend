package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

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

	// when the round of 16 is full, set prediction close and notify users
	app.OnRecordAfterUpdateRequest("draw_slot").Add(func(e *core.RecordUpdateEvent) error {
		drawId := e.Record.GetString("draw_id")
		draw, err := app.Dao().FindRecordById("draw", drawId)
		if err != nil {
			log.Panicln(err)
		}

		size := float64(draw.GetInt("size"))
		r16Round := math.Log2(size) - float64(3)

		log.Println(e.Record)

		if float64(e.Record.GetInt("round")) != r16Round {
			return nil
		}

		filter := fmt.Sprintf(`draw_id="%s"&&round="%d"`, drawId, int(r16Round))
		r16Slots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(r16Slots) != 16 {
			return nil
		}

		predictionClose := time.Now().Add(12 * time.Hour)
		draw.Set("prediction_close", predictionClose)
		if err := app.Dao().SaveRecord(draw); err != nil {
			return err
		}

		return nil
	})

	// when a match result comes in, award points for correct predictions
	app.OnRecordAfterUpdateRequest("draw_slot").Add(func(e *core.RecordUpdateEvent) error {
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

			points := 0

			if strings.Contains(record.GetString("name"), name) && name != "" {
				size := float64(vp.GetInt("size"))
				r16Round := math.Log2(size) - float64(3)

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
			}

			if points != record.GetInt("points") {
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
