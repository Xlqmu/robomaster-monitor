package crawler

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const (
	articleURL        = "https://bbs.robomaster.com/article"
	latestArticleFile = "latest_article.txt"
)

// Article holds the information about a newly found article.
type Article struct {
	Title string
	URL   string
	Href  string // The unique part of the URL used for history comparison
}

// Login is a public function to perform the login sequence.
func Login(ctx context.Context, username, password string) error {
	const passwordTabSelector = `a[data-usagetag="password_login_tab"]`
	const usernameSelector = `input[name="username"]`
	const passwordSelector = `input[type="password"]`
	const loginButtonSelector = `button[data-usagetag="login_button"]`
	const successSelector = `img.user-header.g-avatar`
	const postLoginLoadSelector = `a.articleItem`

	log.Println("Starting login process...")
	err := chromedp.Run(ctx,
		// WARM-UP STEP to stabilize the browser instance in the CI environment.
		chromedp.ActionFunc(func(c context.Context) error {
			log.Println("Performing warm-up navigation to Google...")
			return nil
		}),
		chromedp.Navigate(`https://www.google.com`),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(c context.Context) error {
			log.Println("Warm-up complete. Proceeding to RoboMaster...")
			return nil
		}),

		// Main login sequence
		chromedp.Navigate(articleURL),
		chromedp.WaitVisible(passwordTabSelector),
		chromedp.Click(passwordTabSelector),
		chromedp.WaitVisible(usernameSelector),
		chromedp.SendKeys(usernameSelector, username),
		chromedp.SendKeys(passwordSelector, password),
		chromedp.Click(loginButtonSelector),
		chromedp.WaitVisible(postLoginLoadSelector, chromedp.ByQuery),
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Sleep(1*time.Second),
		chromedp.WaitVisible(successSelector, chromedp.ByQuery),
	)

	if err != nil {
		return fmt.Errorf("automated login failed: %w", err)
	}
	return nil
}

// CheckForUpdate is a public function that checks for a new article.
func CheckForUpdate(ctx context.Context) (*Article, error) {
	log.Println("Checking for new articles...")

	var htmlContent string
	const articleLinkSelector = `a.articleItem`

	err := chromedp.Run(ctx,
		chromedp.Navigate(articleURL),
		chromedp.WaitVisible(articleLinkSelector),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get page content after login: %w", err)
	}
	log.Println("Page content retrieved successfully, now parsing...")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	lastSeenURL, err := loadLatestArticle()
	if err != nil {
		return nil, fmt.Errorf("failed to load history file: %w", err)
	}

	var foundArticle *Article

	doc.Find(articleLinkSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.Find("div.articleItem__titles svg").Length() > 0 {
			log.Printf("Found pinned/official article, skipping: '%s'", s.Find("div.articleItem__title").Text())
			return true
		}
		title := strings.TrimSpace(s.Find("div.articleItem__title").Text())
		href, exists := s.Attr("href")
		if !exists {
			return true
		}

		foundArticle = &Article{
			Title: title,
			URL:   "https://bbs.robomaster.com" + href,
			Href:  href,
		}
		return false
	})

	if foundArticle == nil {
		log.Println("Warning: Could not find any non-pinned articles on the page.")
		return nil, nil
	}

	log.Printf("Latest article on site is: %s ('%s')", foundArticle.Href, foundArticle.Title)

	if foundArticle.Href != lastSeenURL {
		if err := updateLatestArticle(foundArticle.Href); err != nil {
			log.Printf("Warning: Failed to update history file: %v", err)
		}
		return foundArticle, nil
	} else {
		log.Println("No new articles found.")
		return nil, nil
	}
}

func loadLatestArticle() (string, error) {
	if _, err := os.Stat(latestArticleFile); os.IsNotExist(err) {
		return "", nil
	}
	content, err := ioutil.ReadFile(latestArticleFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func updateLatestArticle(latestLink string) error {
	return ioutil.WriteFile(latestArticleFile, []byte(latestLink), 0644)
}
