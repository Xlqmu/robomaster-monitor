package main

import (
	"fmt"
	"log"
	"robomaster-monitor/internal/crawler"
)

func main() {
	url := "https://bbs.robomaster.com/article"
	articles, err := crawler.FetchArticles(url)
	if err != nil {
		log.Fatalf("抓取失败:%v", err)
	}

	fmt.Printf("✅ 抓到文章：%d 条\n\n", len(articles))
	if len(articles) == 0 {
		fmt.Println("注意：未抓到任何条目。已将页面 HTML 保存在 dump.html，请用浏览器打开并检查 DOM")
	}
	for _, a := range articles {
		fmt.Printf("ID: %s\n标题: %s\n作者: %s\n时间: %s\n链接: %s\n\n",
			a.ID, a.Title, a.Author, a.Time, a.Link)
	}
}
