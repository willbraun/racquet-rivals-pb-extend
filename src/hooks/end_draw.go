package hooks

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterEndDrawHook(app core.App) {
	app.OnRecordAfterUpdateSuccess("draw_slot").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("name") == "" {
			return e.Next()
		}

		drawId := e.Record.GetString("draw_id")
		draw, err := app.FindRecordById("draw", drawId)
		if err != nil {
			log.Panicln(err)
		}

		// If there are still empty slots, do not end the draw
		filter := fmt.Sprintf(`draw_id="%s"&&name=""`, drawId)
		emptySlots, err := app.FindRecordsByFilter("draw_slot", filter, "", 1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(emptySlots) > 0 {
			return e.Next()
		}

		// Set draw end date to now if all slots are filled
		draw.Set("end_date", e.Record.GetDateTime("updated"))
		if err := app.Save(draw); err != nil {
			log.Panicln(err)
		}

		return e.Next()
	})
}
