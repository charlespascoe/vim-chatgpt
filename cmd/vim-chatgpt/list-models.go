package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type ListModelsCmd struct{}

func (listmodels *ListModelsCmd) Run(ctx *Context) error {
	client := ctx.Client

	models, err := client.ListModels(ctx)
	if err != nil {
		return err
	}

	var list []string

	for _, model := range models.Models {
		list = append(list, model.ID)
	}

	sort.Strings(list)

	fmt.Println(strings.Join(list, "\n"))

	return nil
}

func printModels(ctx context.Context, client *openai.Client) {
}
