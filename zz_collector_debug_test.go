package main

import (
  "encoding/json"
  "fmt"
  "testing"
)

func TestCollectAIBotPage(t *testing.T) {
  result, err := collectWebPage(CollectWebPageInput{URL: "https://ai-bot.cn/sites/4189.html"})
  if err != nil {
    t.Fatal(err)
  }
  payload, _ := json.MarshalIndent(result, "", "  ")
  fmt.Println(string(payload))
}
