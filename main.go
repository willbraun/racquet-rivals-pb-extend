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
	app.OnRecordAfterUpdateRequest("draw_slot").Add(func(e *core.RecordUpdateEvent) error {
		log.Println("RECORD", e.Record)
		name := e.Record.GetString("name")
		filter := fmt.Sprintf("draw_slot_id=%s", e.Record.Id)

		predictions, err := app.Dao().FindRecordsByFilter("view_predictions", filter, "", -1, 0)
		if err != nil {
			log.Fatal(err)
		}

		for _, p := range predictions {
			record, err := app.Dao().FindRecordById("prediction", p.GetString("id"))
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(record.GetString("id"))

			if strings.Contains(p.GetString("name"), name) {
				// Quarterfinals: 1, Semifinals: 2, Final: 4, Winner: 8
				size := float64(record.GetInt("size"))
				r16Round := math.Log2(size) - float64(4)
				thisRound := float64(record.GetInt("round"))
				points := math.Pow(thisRound-r16Round, 2)
				record.Set("points", points)
				fmt.Println(points)
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
