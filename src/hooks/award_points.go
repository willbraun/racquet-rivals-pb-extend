package hooks

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// When a match result comes in, award points for correct predictions
func RegisterAwardPointsHook(app core.App) {
	app.OnRecordAfterUpdateSuccess("draw_slot").BindFunc(func(e *core.RecordEvent) error {
		name := e.Record.GetString("name")
		round := float64(e.Record.GetInt("round"))
		filter := fmt.Sprintf(`draw_slot_id="%s"`, e.Record.Id)

		view_predictions, err := app.FindRecordsByFilter("view_predictions", filter, "", -1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(view_predictions) == 0 {
			return e.Next()
		}

		for _, vp := range view_predictions {
			record, err := app.FindRecordById("prediction", vp.GetString("id"))
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

			if points == record.GetInt("points") {
				continue
			}

			record.Set("points", points)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return e.Next()
	})
}
