package main

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const transformPrompt = `You are tasked with editing or transforming text and
code. The first message will be an instruction, the second message will be the
code or text. Do not reply with anything other than the output text.`

const generatePrompt = `You are tasked with generating text and code. The
message will be the instructions of what to generate. Do not reply with anything
other than the output text.`

func ApplyEdit(ctx context.Context, client *openai.Client, model, input, instruction string) (string, error) {
	req := openai.ChatCompletionRequest{Model: model}

	if len(input) == 0 {
		req.Messages = append(
			req.Messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: strings.ReplaceAll(generatePrompt, "\n", " "),
			},
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: instruction,
			},
		)
	} else {
		req.Messages = append(
			req.Messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: strings.ReplaceAll(transformPrompt, "\n", " "),
			},
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: instruction,
			},
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: input,
			},
		)
	}

	fmt.Println("Request:", req)

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
