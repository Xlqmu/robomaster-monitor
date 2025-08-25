package crawler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Article表示一条文章信息
type Article struct {
	ID     string
	Title  string
	Link   string
	Author string
	Time   string
}

// FetchArticles 用普通 HTTP 请求抓取页面并解析
// 如果没有抓到任何条目，会把响应 HTML 写到dump.html 以便你调试
func FetchArticles(url string) ([]Article, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %v", err)
	}
	// 常用 header，避免被简单拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("响应错误，状态码: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 解析文档
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %v", err)
	}

	var articles []Article

	// 根据你提供的 DOM 结构，主块可能是 .articleItem__upper 或 .articleItem
	doc.Find("div.articleItem__upper, div.articleItem").Each(func(i int, s *goquery.Selection) {
		// 标题
		title := strings.TrimSpace(s.Find("div.articleItem__title").Text())
		if title == "" { // 备选：查找第一个 <a>
			title = strings.TrimSpace(s.Find("a").First().Text())
		}

		// 链接：优先找 href 包含 /article/ 的 <a>
		href, _ := s.Find(`a[href*="/article/"]`).Attr("href")
		if href == "" {
			// 备选：遍历所有 a标签取第一个包含 /article/
			s.Find("a").EachWithBreak(func(j int, a *goquery.Selection) bool {
				if h, ok := a.Attr("href"); ok && strings.Contains(h, "/article/") {
					href = h
					return false
				}
				return true
			})
		}
		link := href
		if strings.HasPrefix(href, "/") {
			link = "https://bbs.robomaster.com" + href
		}

		author := strings.TrimSpace(s.Find(".articleItem__nickname, .articleItem__author").First().Text())
		datetime := strings.TrimSpace(s.Find(".articleItem__datetime .articleItem__number").First().Text()) // 提取 id（/article/12345 -> 12345）
		id := ""
		if href != "" {
			re := regexp.MustCompile(`/article/(\d+)`)
			if m := re.FindStringSubmatch(href); len(m) >= 2 {
				id = m[1]
			}
		}
		if title != "" {
			articles = append(articles, Article{
				ID:     id,
				Title:  title,
				Link:   link,
				Author: author,
				Time:   datetime,
			})
		}
	})

	// 如果没抓到任何文章，把 HTML dump 出来，便于你在浏览器里查看实际内容
	if len(articles) == 0 {
		_ = os.WriteFile("dump.html", bodyBytes, 0644)
	}
	return articles, nil
}
