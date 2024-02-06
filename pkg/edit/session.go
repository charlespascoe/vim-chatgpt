package edit

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	// Use this to diff text and get ChatGPT to check its work
	"github.com/kylelemons/godebug/diff"

	openai "github.com/sashabaranov/go-openai"
)

//go:embed system-prompt.md
var systemPrompt string

const schema = `
type: object
required:
  - edits
properties:
  edits:
    type: array
    description: "An array of edits to be made to the document. Your response
      MUST ALWAYS be an array of these objects, even if there is only one edit
      to be made."
    items:
      type: object
      description: "An edit to be made to the document."
      required:
        - start
        - end
      properties:
        start:
          type: integer
          description: "The line number from the original input of the first
            line in the range of lines to be replaced."
        end:
          type: integer
          description: "The line number from the original input of the last line
            in the range of lines to be replaced. Make sure this is the correct
            line number, as it is inclusive. When replacing a single line, this
            will be the same as the start. If negative, the replacements will be
            inserted at the position of start line (pushing it down), and no
            lines will be deleted."
        replacement:
          type: array
          description: "The lines to replace the edited lines with. An empty
            array, or omitting this property, will delete the lines."
          items:
            type: string
            description: "A line of text to replace the edited lines with. Do
              not include the line number or Guillemet, nor a newline
              character. You MUST include any leading spaces or tabs for
              indentation."
`

type Session struct {
	filename string
	model    string
	messages []openai.ChatCompletionMessage
	client   *openai.Client
}

func NewSession(client *openai.Client, model, filename string) *Session {
	return &Session{
		filename: filename,
		model:    model,
		messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		},
		client: client,
	}
}

func (s *Session) Apply(ctx context.Context, contents, instructions string) ([]Edit, error) {
	lines := strings.Split(contents, "\n")
	formatted := PrefixLineNums(lines)

	fmt.Println(formatted)

	req := openai.ChatCompletionRequest{
		Model:    s.model,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	req.Messages = make([]openai.ChatCompletionMessage, len(s.messages), len(s.messages)+1)
	copy(req.Messages, s.messages)

	sysPrompt := fmt.Sprintf("The next message is the file contents to edit. The file is called '%s'. The message after it is the user's instructions that you must follow. You MUST provide your response as an array of Edit objects, as described in the schema above. Remember to include indentation spaces/tabs at the start of each line.", s.filename)
	fmt.Println(sysPrompt)

	instructMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: instructions,
	}

	req.Messages = append(
		req.Messages,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: formatted,
		},
		instructMsg,
	)

	for i := 0; i < 3; i++ {
		printJson(req.Messages)
		start := time.Now()
		resp, err := s.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return nil, err
		}

		fmt.Println("Time to get response:", time.Since(start).String())

		data := resp.Choices[0].Message.Content

		fmt.Println(data)

		var edits struct {
			Edits []Edit `json:"edits"`
		}
		var errs []string

		if err = json.Unmarshal([]byte(data), &edits); err != nil {
			fmt.Println("Unable to unmarshal response:", err)
			errs = append(errs, fmt.Sprintf("  Unable to parse JSON response: %s", err))
			// var edit Edit

			// if err = json.Unmarshal([]byte(data), &edit); err != nil {
			// 	fmt.Println("Unable to unmarshal response:", err)
			// } else {
			// 	edits = append(edits, edit)
			// }
		} else if len(edits.Edits) == 0 {
			errs = append(errs, "  No edits provided")
		}

		for i, e := range edits.Edits {
			if err := e.Validate(); err != nil {
				errs = append(errs, fmt.Sprintf("  edits[%d]: %s", i, err))
			}
		}

		if AnyOverlapping(edits.Edits) {
			errs = append(errs, "  Edits must not overlap with each other")
		}

		if len(errs) > 0 {
			req.Messages = append(
				req.Messages,
				resp.Choices[0].Message,
				openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf("Unable to parse response:\n%s\n\nExpected it to be a JSON array of Edit objects that conform to the schema below; please try again.\n%s", strings.Join(errs, "\n"), schema),
				},
			)
			// TODO: Log properly
			fmt.Println("Unable to unmarshal response; retrying")
			fmt.Println(data)
			continue
		}

		s.messages = append(s.messages, instructMsg, resp.Choices[0].Message)

		result := strings.Join(Apply(lines, edits.Edits), "\n")
		fmt.Println("Result:")
		fmt.Println(result)
		d := diff.Diff(contents, result)
		fmt.Println("Diff:")
		fmt.Println(d)

		return edits.Edits, nil
	}

	return nil, fmt.Errorf("failed to get a valid response after 3 attempts")
}

func printJson(arg any) {
	data, err := json.MarshalIndent(arg, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))
}
