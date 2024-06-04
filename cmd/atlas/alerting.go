package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"raidhub/shared/discord"
	"raidhub/shared/monitoring"
	"raidhub/shared/pgcr"

	"golang.org/x/time/rate"
)

var (
	_atlasWebhookURL string
	once             sync.Once
	webhookRl        = rate.NewLimiter(rate.Every(10*time.Second), 5)
)

func getAtlasWebhookURL() string {
	once.Do(func() {
		_atlasWebhookURL = os.Getenv("ATLAS_WEBHOOK_URL")
	})
	return _atlasWebhookURL
}

func handlePanic(r interface{}) {
	content := fmt.Sprintf("<@&%s>", os.Getenv("ALERTS_ROLE_ID"))
	webhook := discord.Webhook{
		Content: &content,
		Embeds: []discord.Embed{{
			Title: "Fatal error in Atlas",
			Color: 10038562, // DarkRed
			Fields: []discord.Field{{
				Name:  "Error",
				Value: fmt.Sprintf("%s", r),
			}},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    discord.CommonFooter,
		}},
	}
	discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	log.Printf("Fatal error in Atlas: %s", r)
}

func sendStartUpAlert() {
	msg := "Info: Starting up..."
	webhook := discord.Webhook{
		Embeds: []discord.Embed{{
			Title:     "Starting up...",
			Color:     3447003, // Blue
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    discord.CommonFooter,
		}},
	}
	discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	log.Println(msg)
}

func logIntervalState(medianLag float64, countWorkers int, percentNotFound float64) {
	webhook := discord.Webhook{
		Embeds: []discord.Embed{{
			Title: "Status Update",
			Color: 9807270, // Gray
			Fields: []discord.Field{{
				Name:  "Lag Behind Head",
				Value: fmt.Sprintf("%1.f seconds", medianLag),
			}, {
				Name:  "404 Percentage",
				Value: fmt.Sprintf("%.3f%%", percentNotFound),
			}, {
				Name:  "Workers Used",
				Value: fmt.Sprintf("%d", countWorkers),
			}},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    discord.CommonFooter,
		}},
	}
	discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	log.Printf("Info: Head is behind by %1.f seconds with %.3f%% not found using %d workers ", medianLag, percentNotFound, countWorkers)
}

func logWorkersStarting(numWorkers int, period int, latestId int64) {
	monitoring.ActiveWorkers.Set(float64(numWorkers))

	webhook := discord.Webhook{
		Embeds: []discord.Embed{{
			Title: "Workers Starting",
			Color: 9807270, // Gray
			Fields: []discord.Field{{
				Name:  "Count",
				Value: fmt.Sprintf("%d", numWorkers),
			}, {
				Name:  "Period",
				Value: fmt.Sprintf("%d", period),
			}, {
				Name:  "Current Instance Id",
				Value: fmt.Sprintf("`%d`", latestId),
			}},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    discord.CommonFooter,
		}},
	}
	discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	log.Printf("Info: %d workers starting at %d", numWorkers, latestId)
}

func logMissedInstance(instanceId int64, startTime time.Time) {
	pgcr.WriteMissedLog(instanceId)

	elapsed := time.Since(startTime).Seconds()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := webhookRl.Wait(ctx)
	if err == nil {
		webhook := discord.Webhook{
			Embeds: []discord.Embed{{
				Title: "Unresolved Instance",
				Color: 15548997, // Red
				Fields: []discord.Field{{
					Name:  "Instance Id",
					Value: fmt.Sprintf("`%d`", instanceId),
				}, {
					Name:  "Time Elapsed",
					Value: fmt.Sprintf("%1.f seconds", elapsed),
				}},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    discord.CommonFooter,
			}},
		}

		discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	}
	log.Printf("Missed PGCR %d after %1.f seconds", instanceId, time.Since(startTime).Seconds())
}

func logMissedInstanceWarning(instanceId int64, startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := webhookRl.Wait(ctx)

	if err == nil {
		webhook := discord.Webhook{
			Embeds: []discord.Embed{{
				Title: "Unresolved Instance (Warning)",
				Color: 15105570, // Orange
				Fields: []discord.Field{{
					Name:  "Instance Id",
					Value: fmt.Sprintf("`%d`", instanceId),
				}, {
					Name:  "Time Elapsed",
					Value: fmt.Sprintf("%1.f seconds", elapsed),
				}},
				Timestamp: time.Now().Format(time.RFC3339),
				Footer:    discord.CommonFooter,
			}},
		}
		discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	}
	log.Printf("Warning: instance id %d has not resolved in %1.f seconds", instanceId, time.Since(startTime).Seconds())
}

func logInsufficentPrivileges(instanceId int64) {
	webhook := discord.Webhook{
		Embeds: []discord.Embed{{
			Title: "InsufficientPrivileges Response",
			Color: 15548997, // Red
			Fields: []discord.Field{{
				Name:  "Instance Id",
				Value: fmt.Sprintf("`%d`", instanceId),
			}},
			Timestamp: time.Now().Format(time.RFC3339),
			Footer:    discord.CommonFooter,
		}},
	}
	discord.SendWebhook(getAtlasWebhookURL(), &webhook)
	log.Printf("Warning: InsufficientPrivileges response for instanceId %d", instanceId)
}
