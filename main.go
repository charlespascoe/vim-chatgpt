package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	conf := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
	conf.HTTPClient.Timeout = 5*time.Second

	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		conf.BaseURL = baseURL
	}

	c := openai.NewClientWithConfig(conf)
	ctx := context.Background()

	chat := NewChat(c, ctx, "You are a helpful assistant. Provide answers using correct Markdown syntax.")

	go func() {
		for err := range chat.Err {
			fmt.Fprintf(os.Stderr, "Chat error: %s\n", err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		var msg Message

		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			fmt.Fprintf(os.Stderr, "Unmarshal error: %s\n", err)
			continue
		}

		chat.UserMessage(msg)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner error: %s\n", err)
		os.Exit(2)
	}
}

type Message struct {
	Text string `json:"text"`
}

type Chat struct {
	Err chan error

	model     string
	messages  []openai.ChatCompletionMessage
	ctx       context.Context
	client    *openai.Client
	strm      *openai.ChatCompletionStream
	userMsgs  chan Message
	startOnce sync.Once
	timeout   time.Duration
	output    io.StringWriter
}

func NewChat(client *openai.Client, ctx context.Context, systemPrompt string) *Chat {
	chat := &Chat{
		model: openai.GPT3Dot5Turbo,
		messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		},
		ctx:      ctx,
		userMsgs: make(chan Message, 1),
		Err:      make(chan error, 5),
		client:   client,
		timeout:  30 * time.Second,
		output:   os.Stdout,
	}

	return chat
}

func (chat *Chat) UserMessage(msg Message) {
	chat.userMsgs <- msg
	chat.startOnce.Do(func() {
		go chat.loop()
	})
}

func (chat *Chat) loop() {
	var cancel context.CancelFunc
	var recv chan string
	var respMsg string

	for {
		select {
		case msg := <-chat.userMsgs:
			if cancel != nil {
				cancel()
				recv = nil
				cancel = nil
			}

			if msg.Text == "" {
				// This was just to stop the existing stream
				continue
			}

			if respMsg != "" {
				chat.messages = append(chat.messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: respMsg,
				})
				respMsg = ""
				chat.output.WriteString("\n\n")
			}

			chat.messages = append(chat.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Text,
			})

			writeQuoted(chat.output, msg.Text)
			chat.output.WriteString("\n")

			var ctx context.Context
			ctx, cancel = context.WithTimeout(chat.ctx, chat.timeout)
			recv = make(chan string, 5)

			go chat.getResponse(ctx, recv)

		case resp, ok := <-recv:
			if !ok {
				// End of stream
				recv = nil
				cancel = nil
				chat.output.WriteString("\n\n")
			}

			respMsg += resp
			chat.output.WriteString(strings.ReplaceAll(resp, "\t", "    "))
		}
	}
}

func (chat *Chat) getResponse(ctx context.Context, recv chan string) {
	defer close(recv)

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		// MaxTokens: 20,
		Messages: chat.messages,
		Stream:   true,
	}

	strm, err := chat.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		chat.Err <- err
		return
	}

	for {
		response, err := strm.Recv()
		if errors.Is(err, io.EOF) {
			return
		}

		if err != nil {
			chat.Err <- err
			return
		}

		select {
		case recv <- response.Choices[0].Delta.Content:
			// Fine
		default:
			chat.Err <- errors.New("recv chan blocked")
		}
	}
}

func writeQuoted(writer io.StringWriter, str string) {
	for _, line := range strings.Split(str, "\n") {
		if len(line) > 0 {
			writer.WriteString("> " + line + "\n")
		} else {
			writer.WriteString(">\n")
		}
	}
}
