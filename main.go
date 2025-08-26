package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	// Make sure this is the correct module path from your go.mod file
	"robomaster-monitor/internal/crawler"
	"robomaster-monitor/internal/notifier"
)

const (
	// ==========================================================
	// ===> 在这里修改频率 (Change the frequency here) <===
	// ==========================================================
	checkInterval = 15 * time.Minute
)

func main() {
	// --- CONFIGURATION FOR LOCAL TESTING ---
	const username = "18193854081"
	const password = "nzh040911@"
	const webhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/eeab5932-7217-4276-91f8-4ea5c344c7e9"

	if username == "YOUR_USERNAME_HERE" || password == "YOUR_PASSWORD_HERE" {
		log.Fatal("Error: Please fill in your username and password in main.go for local testing.")
	}
	if webhookURL == "YOUR_FEISHU_WEBHOOK_URL_HERE" {
		log.Println("Warning: Feishu Webhook URL is not provided, notifications will be skipped.")
	}
	// --- END OF CONFIGURATION ---

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("start-maximized", true),
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

	// This loop makes the program run continuously.
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		runPipeline(ctx, webhookURL)
		log.Printf("Next check in %v.", checkInterval)
		<-ticker.C
	}
}

// runPipeline executes one cycle of the check-and-notify process.
func runPipeline(ctx context.Context, webhookURL string) {
	newArticle, err := crawler.CheckForUpdate(ctx)
	if err != nil {
		log.Printf("Error during crawler check: %v", err)
		return
	}

	if newArticle != nil {
		log.Println("New article found, sending Feishu notification...")
		if webhookURL != "" && webhookURL != "YOUR_FEISHU_WEBHOOK_URL_HERE" {
			if err := notifier.Send(newArticle.Title, newArticle.URL, webhookURL); err != nil {
				log.Printf("Error sending Feishu notification: %v", err)
			}
		}
	}

	log.Println("Check complete.")
}
