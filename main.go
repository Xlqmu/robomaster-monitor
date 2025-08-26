package main

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	log.Println("Starting final diagnostic test...")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36`),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 3*time.Minute) // 3 minute timeout for the test
	defer cancel()

	var buf []byte

	// --- Test 1: Navigate to Google ---
	log.Println("Attempting to navigate to https://www.google.com ...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(`https://www.google.com`),
		chromedp.Sleep(5*time.Second), // Wait for page to render
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		log.Fatalf("Fatal: Failed to navigate to Google: %v", err)
	}
	if err := ioutil.WriteFile("google_test.png", buf, 0644); err != nil {
		log.Fatalf("Fatal: Failed to save Google screenshot: %v", err)
	}
	log.Println("Success! Screenshot saved as google_test.png.")

	// --- Test 2: Navigate to RoboMaster ---
	log.Println("Attempting to navigate to https://bbs.robomaster.com/article ...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(`https://bbs.robomaster.com/article`),
		chromedp.Sleep(15*time.Second), // Give it extra time
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		// This is the expected failure point if the site is blocking us
		log.Printf("Error: Failed to navigate to RoboMaster: %v. This likely confirms an IP block.", err)
		// We save the (likely blank) screenshot anyway for analysis
	}
	if err := ioutil.WriteFile("robomaster_test.png", buf, 0644); err != nil {
		log.Fatalf("Fatal: Failed to save RoboMaster screenshot: %v", err)
	}
	log.Println("RoboMaster navigation step finished. Screenshot saved as robomaster_test.png.")
	log.Println("Diagnostic complete.")
}
