package crawler

import (
	"context"
	"fmt"
	"log"
	"robomaster-monitor/internal/storage"
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
	Title  string
	URL    string
	Href   string // The unique part of the URL used for history comparison
	Author string
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
		// Main login sequence (warm-up step has been removed)
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

// CheckForUpdate
func CheckForUpdate(ctx context.Context) ([]storage.Article, error) {
	log.Println("🔍 检查新文章...")

	var htmlContent string
	const articleLinkSelector = `a.articleItem`

	err := chromedp.Run(ctx,
		chromedp.Navigate(articleURL),
		chromedp.WaitVisible(articleLinkSelector),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("获取页面内容失败: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	var newArticles []storage.Article
	var processedCount int

	doc.Find(articleLinkSelector).Each(func(i int, s *goquery.Selection) {
		// 跳过置顶文章
		if s.Find("div.articleItem__titles svg").Length() > 0 {
			log.Printf("⏭️ 跳过置顶/官方文章: '%s'", s.Find("div.articleItem__title").Text())
			return
		}

		// only process the first 10 articles
		if processedCount >= 10 {
			return
		}
		processedCount++

		title := strings.TrimSpace(s.Find("div.articleItem__title").Text())
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		author := strings.TrimSpace(s.Find(".articleItem__info-author").Text())
		category := strings.TrimSpace(s.Find(".articleItem__category").Text())
		postedTime := strings.TrimSpace(s.Find(".articleItem__info-time").Text())

		fullURL := "https://bbs.robomaster.com" + href

		// check if the article exists in the database
		exists, err := storage.ArticleExists(fullURL)
		if err != nil {
			log.Printf("⚠️ 检查文章存在性失败: %v", err)
			return
		}

		if !exists {
			newArticle := storage.Article{
				Title:    title,
				URL:      fullURL,
				Author:   author,
				Category: category,
				PostedAt: postedTime,
				Notified: false,
			}

			id, err := storage.SaveArticle(&newArticle)
			if err != nil {
				log.Printf("⚠️ 保存文章失败: %v", err)
				return
			}

			newArticle.ID = id
			newArticles = append(newArticles, newArticle)
		}
	})

	if len(newArticles) > 0 {
		log.Printf("🆕 发现 %d 篇新文章", len(newArticles))
	} else {
		log.Println("✅ 没有发现新文章")
	}

	return newArticles, nil
}
