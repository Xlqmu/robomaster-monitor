package main

// package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/BurntSushi/toml"
)

type FeishuConfig struct {
	WebhookURL string `toml:"webhook_url"`
}

type Config struct {
	Feishu FeishuConfig `toml:"feishu"`
}

func sendMessage() {
	var config Config
	_, err := toml.DecodeFile("configs/param.toml", &config)
	if err != nil {
		panic(err)
	}

	webhookURL := config.Feishu.WebhookURL

	msg := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": "叮!RM论坛又有新的开源了🚀",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)
}

// test
func main() {
	sendMessage()
}
