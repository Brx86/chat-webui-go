package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"

	"github.com/sashabaranov/go-openai"
	"resty.dev/v3"
)

type ModelListData struct {
	Data []struct {
		Id string
	}
}

type ChatData struct {
	Message      json.RawMessage
	MessageText  string
	MessagePart  []openai.ChatMessagePart
	Conversation []struct {
		Content string
		Role    string
	}
	Model         string
	SystemContent string
	Parameters    struct {
		Temperature float32
	}
	IsDeepQueryMode   bool
	StartTag          string
	AssistantResponse string
}

var (
	preloadModels []string
	openaiClient  *openai.Client
)

func initModels() {
	if cfg.ApiKey == "" || cfg.BaseUrl == "" {
		log.Println("配置不完整，不加载 modelList")
		return
	}
	fetchModels()
	initClient()
}

func initClient() {
	config := openai.DefaultConfig(cfg.ApiKey)
	config.BaseURL = cfg.BaseUrl
	openaiClient = openai.NewClientWithConfig(config)
}

func fetchModels() {
	client := resty.New()
	defer client.Close()

	modelListData := new(ModelListData)
	resp, err := client.SetBaseURL(cfg.BaseUrl).
		// SetProxy("socks5://192.168.50.3:7891").
		SetAuthToken(cfg.ApiKey).
		R().
		SetResult(modelListData).
		Get("/models")
	if err != nil || resp.StatusCode() != 200 {
		log.Println("加载 modelList 失败: ", err)
		return
	}
	preloadModels = make([]string, len(modelListData.Data))
	for i, d := range modelListData.Data {
		preloadModels[i] = d.Id
	}
	// preloadModels = slices.DeleteFunc(preloadModels, func(s string) bool { return !strings.Contains(s, "2.5") })
}

func newMsg(role, content string, parts []openai.ChatMessagePart) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:         role,
		Content:      content,
		MultiContent: parts,
	}
}

func makeMsgs(d *ChatData) []openai.ChatCompletionMessage {
	msgs := make([]openai.ChatCompletionMessage, 0, len(d.Conversation)+2)
	if d.SystemContent != "" {
		msgs = append(msgs, newMsg("system", d.SystemContent, nil))
	}
	for _, c := range d.Conversation {
		msgs = append(msgs, newMsg(c.Role, c.Content, nil))
	}
	if d.MessageText != "" {
		msgs = append(msgs, newMsg("user", d.MessageText, nil))
	}
	if len(d.MessagePart) > 0 {
		msgs = append(msgs, newMsg("user", "", d.MessagePart))
	}
	return msgs
}

func generate(ch chan<- string, d *ChatData) {
	defer close(ch)
	stream, err := openaiClient.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       d.Model,
			Messages:    makeMsgs(d),
			Temperature: d.Parameters.Temperature,
			Stream:      true,
		},
	)
	if err != nil {
		ch <- err.Error()
		log.Println(err)
		return
	}
	defer stream.Close()

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			ch <- err.Error()
			return
		}
		ch <- resp.Choices[0].Delta.Content
	}
}

func genTitle(d *ChatData) string {
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{Model: d.Model, Messages: []openai.ChatCompletionMessage{
			newMsg("system", "你是一位生成对话短标题的专家。请根据user的消息和assistant的回复，为对话生成一个非常简短的标题（最多8个词）。标题应捕捉对话的主要主题或目的。只回复标题，不带引号、额外文本或任何其他字符。", nil),
			newMsg("user", "User message:"+d.MessageText+"\n\nAssistant response:"+d.AssistantResponse, nil),
		}},
	)
	if err == nil && len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}
	return d.MessageText[:10]
}
