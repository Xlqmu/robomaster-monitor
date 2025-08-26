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

	log.Println("Navigating to article page to trigger login redirect...")
	err := chromedp.Run(ctx,
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
// It returns a pointer to an Article if a new one is found, otherwise it returns nil.
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

	// Find all article items and loop through them to find the first non-pinned one.
	doc.Find(articleLinkSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		// If the article item has an SVG in its title area, it's pinned/official.
		if s.Find("div.articleItem__titles svg").Length() > 0 {
			log.Printf("Found pinned/official article, skipping: '%s'", s.Find("div.articleItem__title").Text())
			return true // Continue to the next item
		}

		// This is the first non-pinned article. Extract its details.
		title := strings.TrimSpace(s.Find("div.articleItem__title").Text())
		href, exists := s.Attr("href")
		if !exists {
			log.Println("Warning: Found an article item with no link, skipping.")
			return true // Continue to the next item
		}

		foundArticle = &Article{
			Title: title,
			URL:   "https://bbs.robomaster.com" + href,
			Href:  href,
		}
		// We have found the first valid article, so we stop the loop.
		return false
	})

	// Now that the loop is finished, process the article we found (if any).
	if foundArticle == nil {
		log.Println("Warning: Could not find any non-pinned articles on the page.")
		return nil, nil
	}

	log.Printf("Latest article on site is: %s ('%s')", foundArticle.Href, foundArticle.Title)

	if foundArticle.Href != lastSeenURL {
		if err := updateLatestArticle(foundArticle.Href); err != nil {
			log.Printf("Warning: Failed to update history file: %v", err)
		}
		// Return the found article because it's new.
		return foundArticle, nil
	} else {
		log.Println("No new articles found.")
		// The latest article is the same as the one we've seen before. Return nil.
		return nil, nil
	}
}

// loadLatestArticle and updateLatestArticle are private helper functions for this package.
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
