package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	//"git.server.lan/pkg/config/realtimeconfig"

	"git.server.lan/pkg/config/realtimeconfig"
	"github.com/psevdocoder/gentleman-ping-bot/internal/apiclient"
	"github.com/psevdocoder/gentleman-ping-bot/internal/config"
	"github.com/psevdocoder/gentleman-ping-bot/internal/curlparse"
	"github.com/psevdocoder/gentleman-ping-bot/internal/sender"
	cron "github.com/psevdocoder/gentleman-ping-bot/pkg/cron"
)

func main() {
	if err := realtimeconfig.StartWatching(); err != nil {
		log.Fatal(err)
	}

	curlFilePathRaw, err := config.GetValue(config.CurlFile)
	if err != nil {
		log.Fatal(err)
	}

	curlFilePath, err := curlFilePathRaw.String()
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(curlFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	curlRaw, err := io.ReadAll(file)

	parser := curlparse.NewParser(string(curlRaw))
	client := apiclient.NewClient()
	senderJob, err := sender.NewSendMessageJob(parser, client)
	if err != nil {
		log.Fatal(err)
	}

	cronManager := cron.NewCronManager()
	cronManager.Start()

	ctx := context.Background()

	senderCronSpecraw, err := config.GetValue(config.CronExpr)
	if err != nil {
		log.Fatal(err)
	}

	senderCronSpecStr, err := senderCronSpecraw.String()
	if err != nil {
		log.Fatal(err)
	}

	config.Watch(config.CronExpr, func(newValue, oldValue realtimeconfig.Value) {
		newCronSpecStr, err := newValue.String()
		if err != nil {
			log.Println("Failed to parse cron expression in live config:", err)
			return
		}

		oldCronSpecStr, err := oldValue.String()
		if err != nil {
			log.Println("Failed to parse old cron expression in live config:", err)
			return
		}

		if err := cronManager.RemoveTask(senderJob.Name()); err != nil {
			log.Println("Failed to remove task in live config:", err)
			return
		}

		if err := cronManager.AddTask(ctx, newCronSpecStr, senderJob); err != nil {
			log.Println("Failed to add task in live config:", err)
			return
		}

		log.Printf("Changed sender cron from %s to %s", oldCronSpecStr, newCronSpecStr)
	})

	if err := cronManager.AddTask(ctx, senderCronSpecStr, senderJob); err != nil {
		log.Fatal(err)
	}

	syscallCh := make(chan os.Signal, 1)
	signal.Notify(syscallCh, syscall.SIGINT, syscall.SIGTERM)
	<-syscallCh

	if err := cronManager.Stop(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Shutting down...")
}
