package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	// Make sure this is the correct module path from your go.mod file
	"robomaster-monitor/internal/crawler"
	"robomaster-monitor/internal/notifier"
)

func main() {
	// --- Reads configuration securely from environment variables ---
	username := os.Getenv("DJI_USERNAME")
	password := os.Getenv("DJI_PASSWORD")
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")

	if username == "" || password == "" {
		log.Fatal("Error: DJI_USERNAME and DJI_PASSWORD environment variables must be set.")
	}
	if webhookURL == "" {
		log.Println("Warning: FEISHU_WEBHOOK_URL is not set. Notifications will be skipped.")
	}
	// --- End of Configuration ---

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// This MUST be true for GitHub Actions.
		chromedp.Flag("headless", true),
		// Required flags for running in a Linux/container environment.
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36`),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := crawler.Login(ctx, username, password); err != nil {
		log.Fatalf("Fatal: Initial login failed: %v", err)
	}
	log.Println("Login successful, session is active.")

	// For a scheduled task, we only need to check once per run.
	newArticle, err := crawler.CheckForUpdate(ctx)
	if err != nil {
		log.Fatalf("Fatal: Error during check: %v", err)
	}

	// If the crawler found a new article, call the notifier.
	if newArticle != nil {
		log.Println("New article found, sending Feishu notification...")
		if webhookURL != "" {
			if err := notifier.Send(webhookURL, newArticle.Title, newArticle.URL); err != nil {
				log.Printf("Error sending Feishu notification: %v", err)
			}
		}
	}

	log.Println("Check complete.")
}
