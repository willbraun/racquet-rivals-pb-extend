package main

import (
	"fmt"
	"log"
	"math"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
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
		if e.Record.GetString("name") == "" {
			return nil
		}

		drawId := e.Record.GetString("draw_id")
		draw, err := app.Dao().FindRecordById("draw", drawId)
		if err != nil {
			log.Panicln(err)
		}

		if !draw.GetDateTime("prediction_close").IsZero() {
			return nil
		}

		size := float64(draw.GetInt("size"))
		r16Round := math.Log2(size) - float64(3)

		if float64(e.Record.GetInt("round")) != r16Round {
			return nil
		}

		filter := fmt.Sprintf(`draw_id="%s"&&round="%d"&&name!=""`, drawId, int(r16Round))
		r16FilledSlots, err := app.Dao().FindRecordsByFilter("draw_slot", filter, "", -1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(r16FilledSlots) != 16 {
			return nil
		}

		predictionClose := time.Now().Add(12 * time.Hour)
		draw.Set("prediction_close", predictionClose)
		if err := app.Dao().SaveRecord(draw); err != nil {
			return err
		}

		name := draw.GetString("name")
		event := strings.ReplaceAll(draw.GetString("event"), "'", "")
		year := draw.GetInt("year")
		title := fmt.Sprintf("%s %s %d", name, event, year)
		slug := strings.ToLower(strings.ReplaceAll(strings.Join([]string{name, event, strconv.Itoa(year)}, "-"), " ", "-")) + "-" + drawId

		users, err := app.Dao().FindRecordsByFilter("user", `email!=""`, "", -1, 0)
		for _, user := range users {
			message := &mailer.Message{
				From: mail.Address{
					Address: app.Settings().Meta.SenderAddress,
					Name:    app.Settings().Meta.SenderName,
				},
				To:      []mail.Address{{Address: user.GetString("email")}},
				Subject: "Time to make your picks!",
				HTML:    fmt.Sprintf(`The Round of 16 is ready to go for: <b>%s</b>. You have 12 hours to make your picks for ALL of the remaining matches, good luck!<br><br><a href="https://racquetrivals.com/draw/%s">Racquet Rivals - %s</a>`, title, slug, title),
			}

			app.NewMailClient().Send(message)
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

			if points == record.GetInt("points") {
				continue
			}

			record.Set("points", points)
			if err := app.Dao().SaveRecord(record); err != nil {
				return err
			}
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
