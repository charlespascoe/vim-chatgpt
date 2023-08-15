package main

import (
	"context"
	"errors"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const transformPrompt = `You are tasked with editing or transforming text and
code. The first message will be an instruction, the second message will be the
code or text. Do not reply with anything other than the output text.`

const generatePrompt = `You are tasked with generating text and code. The
message will be the instructions of what to generate. Do not reply with anything
other than the output text.`

func ApplyEdit(ctx context.Context, client *openai.Client, model, input, instruction string, output io.StringWriter) error {
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

	strm, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}

	for {
		resp, err := strm.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		output.WriteString(resp.Choices[0].Delta.Content)
	}

	return nil
}
