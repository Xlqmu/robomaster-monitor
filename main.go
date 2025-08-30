package main

import (
	"context"
	"log"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/chromedp/chromedp"
	"github.com/fsnotify/fsnotify"

	"robomaster-monitor/internal/crawler"
	"robomaster-monitor/internal/notifier"
)

const (
	// ==========================================
	// ===> 在这里修改频率 (Change the frequency here) <===
	// ==========================================================
	checkInterval = 5 * time.Minute // 5min抓取一次不会出发滑块验证，具体阈值可自测
	configFile    = "config/param.toml"
)

type Config struct {
	DJI struct {
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"dji"`

	Feishu struct {
		WebhookURL string `toml:"webhook_url"`
	} `toml:"feishu"`

	Browser struct {
		Headless           bool   `toml:"headless"`
		NoSandbox          bool   `toml:"no_sandbox"`
		DisableGPU         bool   `toml:"disable_gpu"`
		DisableDevShmUsage bool   `toml:"disable_dev_shm_usage"`
		UserAgent          string `toml:"user_agent"`
	} `toml:"browser"`
}

var config Config

// 加载配置文件
func loadConfig(path string) {
	if _, err := toml.DecodeFile(path, &config); err != nil {
		log.Fatalf("❌  读取配置文件失败: %v", err)
	}
	log.Println("✅  配置文件加载成功")
}

// 启动热加载监听(支持运行时需盖参数)
func watchConfig(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("🔄  检测到配置文件修改，重新加载配置...")
					loadConfig(path)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("❌  配置文件监听错误:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal("添加配置文件监听失败:", err)
	}
}

// 执行一次任务流程
func runPipeline() {
	username := config.DJI.Username
	password := config.DJI.Password
	webhookURL := config.Feishu.WebhookURL

	if username == "" || password == "" {
		log.Println("⚠️ 缺少用户名或密码，跳过执行")
		return
	}

	// 浏览器启动参数
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", config.Browser.Headless),
		chromedp.Flag("no-sandbox", config.Browser.NoSandbox),
		chromedp.Flag("disable-gpu", config.Browser.DisableGPU),
		chromedp.Flag("disable-dev-shm-usage", config.Browser.DisableDevShmUsage),
		chromedp.UserAgent(config.Browser.UserAgent),
	)

	pipelineCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	allocCtx, cancel := chromedp.NewExecAllocator(pipelineCtx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 登录
	if err := crawler.Login(taskCtx, username, password); err != nil {
		log.Printf("❌  登录失败: %v", err)
		return
	}
	log.Println("✅  登录成功")

	// 检查更新
	newArticle, err := crawler.CheckForUpdate(taskCtx)
	if err != nil {
		log.Printf("❌  检查更新失败: %v", err)
		return
	}

	// 有新文章则通知
	if newArticle != nil {
		log.Println("🆕  检测到新文章，发送飞书通知...")
		if webhookURL != "" {
			if err := notifier.Send(webhookURL, newArticle.Title, newArticle.URL); err != nil {
				log.Printf("❌  飞书通知失败: %v", err)
			} else {
				log.Println("📨  飞书通知发送成功")
			}
		}
	}

	log.Println("✅  本次检查完成")
}
func main() {
	log.Println("🚀  启动 RoboMaster Monitor...")

	// 1. 加载一次配置
	loadConfig(configFile)

	// 2. 启动热加载监听
	watchConfig(configFile)

	// 3. 启动定时任务
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 立即运行一次
	runPipeline()

	// 循环执行
	for {
		log.Printf("⏳  距离下次检测还有 %v ...", checkInterval)
		<-ticker.C
		runPipeline()
	}
}
