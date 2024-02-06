package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	openai "github.com/sashabaranov/go-openai"
)

type Context struct {
	context.Context
	Client *openai.Client
}

type CLI struct {
	Chat       ChatCmd       `kong:"cmd,help='Start a Chat with the assistant.'"`
	Edit       EditCmd       `kong:"cmd,help='Perform an edit/generation operation on stdin text.'"`
	ListModels ListModelsCmd `kong:"cmd,help='Fetch and list all available models.'"`
}

const scanBufSize = 1024 * 1024

func main() {
	conf := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))

	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		conf.BaseURL = baseURL
	}

	c := openai.NewClientWithConfig(conf)

	var args CLI
	ctx := kong.Parse(&args)

	err := ctx.Run(&Context{
		Context: context.Background(),
		Client:  c,
	})

	ctx.FatalIfErrorf(err)
}
