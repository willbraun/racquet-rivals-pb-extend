package main

import (
	"fmt"
	"log"
	"math"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func bindAppHooks(app core.App) {
	// When the round of 16 is full, set prediction close and notify users
	app.OnRecordAfterUpdateSuccess("draw_slot").BindFunc(func(e *core.RecordEvent) error {
		// Set prediction close
		if e.Record.GetString("name") == "" {
			return e.Next()
		}

		drawId := e.Record.GetString("draw_id")
		draw, err := app.FindRecordById("draw", drawId)
		if err != nil {
			log.Panicln(err)
		}

		if !draw.GetDateTime("prediction_close").IsZero() {
			return e.Next()
		}

		size := float64(draw.GetInt("size"))
		r16Round := math.Log2(size) - float64(3)

		if float64(e.Record.GetInt("round")) != r16Round {
			return e.Next()
		}

		filter := fmt.Sprintf(`draw_id="%s"&&round="%d"&&name!=""`, drawId, int(r16Round))
		r16FilledSlots, err := app.FindRecordsByFilter("draw_slot", filter, "", -1, 0)
		if err != nil {
			log.Panicln(err)
		}

		if len(r16FilledSlots) != 16 {
			return e.Next()
		}

		predictionClose := time.Now().Add(12 * time.Hour)
		draw.Set("prediction_close", predictionClose)
		if err := app.Save(draw); err != nil {
			return err
		}

		// Send email to notify users
		name := draw.GetString("name")
		event := strings.ReplaceAll(draw.GetString("event"), "'", "")
		year := draw.GetInt("year")
		title := fmt.Sprintf("%s %s %d", name, event, year)
		slug := strings.ToLower(strings.ReplaceAll(strings.Join([]string{name, event, strconv.Itoa(year)}, "-"), " ", "-")) + "-" + drawId

		users, err := app.FindRecordsByFilter("user", `email!=""`, "", -1, 0)
		if err != nil {
			return err
		}

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

		return e.Next()
	})

	// When a match result comes in, award points for correct predictions
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

func main() {
	app := pocketbase.New()

	bindAppHooks(app)

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
