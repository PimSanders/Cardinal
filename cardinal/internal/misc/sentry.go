package misc

import (
	"fmt"

	"github.com/getsentry/sentry-go"

	"Cardinal/internal/conf"
	"Cardinal/internal/utils"
)

const sentryDSN = "https://08a91604e4c9434ab6fdc6369ee577d7@o424435.ingest.sentry.io/5356242"

func Sentry() {
	cardinalVersion := utils.VERSION
	cardinalCommitSHA := utils.COMMIT_SHA

	if !conf.App.EnableSentry {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{IPAddress: "{{auto}}"})
	})

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: sentryDSN,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			event.Tags["cardinal_version"] = cardinalVersion
			event.Release = cardinalCommitSHA
			return event
		},
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}

	// greeting
	sentry.CaptureMessage("Hello " + cardinalVersion)
}
