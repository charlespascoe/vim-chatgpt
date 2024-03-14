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

type ChatCmd struct {
	Model        string `placeholder:"MODEL" help:"The model to use. See list-models to see options."`
	SystemPrompt string `help:"The initial hidden system prompt to start the chat."`
	Wrap         int    `help:"Maximum number of columns of the output."`
	TabWidth     int    `help:"Number of columns per tab. Only used when wrapping output." default:"4"`
	ShowPrompt   bool   `help:"Show the system prompt in the output."`
}

func (c *ChatCmd) Run(ctx *Context) error {
	var output io.StringWriter = os.Stdout

	if c.Wrap > 0 {
		output = NewMarkdownWriter(os.Stdout, c.Wrap, c.TabWidth)
	}

	if c.Model == "" {
		c.Model = openai.GPT3Dot5Turbo
	}

	if c.SystemPrompt == "" {
		c.SystemPrompt = "You are a helpful assistant. Provide answers using correct Markdown syntax."
	}

	chat := NewChat(
		ctx,
		ctx.Client,
		output,
		c.Model,
		c.SystemPrompt,
	)

	go func() {
		for err := range chat.Err {
			fmt.Fprintf(os.Stderr, "Chat error: %s\n", err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, scanBufSize), scanBufSize)

	output.WriteString(fmt.Sprintf("# Chat using %s\n\n", c.Model))

	if c.ShowPrompt {
		writeQuoted(output, "System Prompt: "+c.SystemPrompt)
		output.WriteString("\n\n")
	}

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

	return nil
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
	userMsgs  chan Message
	startOnce sync.Once
	timeout   time.Duration
	output    io.StringWriter
}

func NewChat(ctx context.Context, client *openai.Client, output io.StringWriter, model, systemPrompt string) *Chat {
	chat := &Chat{
		model: model,
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
		timeout:  2 * time.Minute,
		output:   output,
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
				// TODO: Handle new lines when aborting a previous response
				// chat.output.WriteString("\n\n")
			}

			chat.messages = append(chat.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Text,
			})

			writeQuoted(chat.output, msg.Text)
			chat.output.WriteString("\n\n")

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
		Model:    chat.model,
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
		case <-ctx.Done():
			return
		}
	}
}

func writeQuoted(writer io.StringWriter, str string) {
	writer.WriteString("> ")

	var output io.StringWriter = NewReplaceWriter(writer, "\n", "\n> ")

	if mdw, ok := writer.(*MarkdownWriter); ok {
		output = NewMarkdownWriter(output, mdw.MaxLen()-2, mdw.TabWidth())
		defer mdw.Flush()
	}

	output.WriteString(strings.TrimSpace(str))
}
