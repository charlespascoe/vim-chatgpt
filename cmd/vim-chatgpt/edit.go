package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charlespascoe/vim-chatgpt/pkg/edit"

	openai "github.com/sashabaranov/go-openai"
)

const transformPrompt = `You are tasked with editing or transforming text and
code. The first message will be an instruction, the second message will be the
code or text. Do not reply with anything other than the output text.`

const generatePrompt = `You are tasked with generating text and code. The
message will be the instructions of what to generate. Do not reply with anything
other than the output text.`

type EditCmd struct {
	Model        string `kong:"short='m',placeholder='MODEL',help='The model to use. See list-models to see options.'"`
	Instructions string `kong:"arg,required,help='The instructions to use.'"`
	Filename     string `kong:"arg,optional,help='The filename to use.'"`
}

func (cmd *EditCmd) Run(ctx *Context) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read stdin error: %s\n", err)
		os.Exit(2)
	}

	if cmd.Model == "" {
		cmd.Model = openai.GPT4TurboPreview
	}

	fmt.Println("Using model:", cmd.Model)

	sess := edit.NewSession(ctx.Client, cmd.Model, cmd.Filename)

	edits, err := sess.Apply(ctx, string(input), cmd.Instructions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Edit error: %s\n", err)
		os.Exit(2)
	}

	data, err := json.MarshalIndent(edits, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON error: %s\n", err)
		os.Exit(2)
	}

	fmt.Println(string(data))

	// err = ApplyEdit(
	// 	ctx,
	// 	ctx.Client,
	// 	cmd.Model,
	// 	string(input),
	// 	cmd.Instructions,
	// 	os.Stdout,
	// )
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Edit error: %s\n", err)
	// 	os.Exit(2)
	// }

	return nil
}

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
