package paddle_webhooks

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/router"
)

type WebhookErrorContext struct {
	App              core.App
	Event            *core.RequestEvent
	Route            string
	Message          string
	RequestBodyBytes []byte
	Error            error
	StatusCode       int
}

func HandleWebhookError(ctx WebhookErrorContext) *router.ApiError {
	NotifySelfWebhookFailure(ctx)
	switch ctx.StatusCode {
	case 400:
		return ctx.Event.BadRequestError(ctx.Message, nil)
	case 401:
		return ctx.Event.UnauthorizedError(ctx.Message, nil)
	case 403:
		return ctx.Event.ForbiddenError(ctx.Message, nil)
	case 404:
		return ctx.Event.NotFoundError(ctx.Message, nil)
	case 429:
		return ctx.Event.TooManyRequestsError(ctx.Message, nil)
	}

	return ctx.Event.InternalServerError(ctx.Message, nil)
}

// NotifySelfWebhookFailure sends an email notification about webhook handling failures
// with strongly typed request body
func NotifySelfWebhookFailure(ctx WebhookErrorContext) {
	message := &mailer.Message{
		From: mail.Address{
			Address: ctx.App.Settings().Meta.SenderAddress,
			Name:    ctx.App.Settings().Meta.SenderName,
		},
		To:      []mail.Address{{Address: "williamhbraun1@gmail.com"}},
		Subject: "Racquet Rivals - Internal Error",
		HTML: fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<style>
					body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; }
					.container { padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
					.header { background-color: #f44336; color: white; padding: 10px 20px; border-radius: 5px 5px 0 0; }
					.content { padding: 20px; background-color: #f9f9f9; font-size: 18px; }
					.message { margin-bottom: 20px; padding: 15px; background-color: white; border-left: 4px solid #f44336; }
					.request { background-color: #f8f8f8; padding: 10px; border-radius: 4px; overflow-x: auto; font-size: 14px; color: #333; }
					.details { background-color: #ebebeb; padding: 15px; font-family: monospace; overflow-wrap: break-word; }
					.footer { font-size: 12px; text-align: center; margin-top: 20px; color: #777; }
				</style>
			</head>
			<body>
				<div class="container">
					<div class="header">
						<h2>Racquet Rivals System Alert</h2>
					</div>
					<div class="content">
						<p>An error has occurred that requires attention:</p>
						<div class="message">
							<strong>Error Message:</strong>
							<p>%s</p>
						</div>
						<div class="request">
							<strong>Request Body:</strong>
							<pre>%s</pre>
						</div>
						<div class="details">
							<strong>Technical Details:</strong>
							<p>%s</p>
						</div>
					</div>
					<div class="footer">
						<p>This is an automated message from the Racquet Rivals system.</p>
						<p>Time: %s</p>
					</div>
				</div>
			</body>
			</html>
			`, ctx.Message, string(ctx.RequestBodyBytes), ctx.Error.Error(), time.Now().Format("Jan 02, 2006 15:04:05 MST")),
	}

	ctx.App.NewMailClient().Send(message)
}
