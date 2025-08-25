# robomaster-monitor

## 项目架构

``` bash
robomaster-monitor/
├── cmd/
│   └── main.go              # 程序入口：执行抓取、比对、通知流程
├── internal/
│   ├── crawler/
│   │    └── fetch.go        # 抓取文章列表
│   ├── parser/
│   │    └── parse.go        # 解析抓到的 HTML 获取文章信息
│   ├── storage/
│   │    └── storage.go      # 本地存储已通知文章（文件或 SQLite）
│   └── notifier/
│        └── feishu.go       # 飞书机器人通知
├── configs/
│   └── config.yaml          # 配置：论坛 URL、Webhook Key、存储路径等
├── .github/
│   └── workflows/
│       └── monitor.yml      # GitHub Actions 配置
├── go.mod
├── README.md
└── Dockerfile               # 可选：打包成容器镜像
```
