package edit

import (
	"errors"
	"fmt"
	"strings"
	"math"
	"sort"
)

type Edit struct {
	Start       int      `json:"start"`
	End         int      `json:"end"`
	Replacement []string `json:"replacement"`
}
// Validate returns an error if any of the fields are invalid.
func (e *Edit) Validate() error {
	if e.Start <= 0 {
		return errors.New("start is required and must be greater than 0")
	}

	if e.End > 0 && e.End < e.Start {
		return errors.New("end must be either zero or not less than start")
	}

	for _, s := range e.Replacement {
		if strings.Contains(s, "\n") {
			return errors.New("replacement cannot contain newline characters")
		}
	}

	return nil
}

func PrefixLineNums(lines []string) string {
	var str strings.Builder

	digits := int(math.Log10(float64(len(lines)))) + 1

	for i, line := range lines {
		str.WriteString(fmt.Sprintf("%0*d»", digits, i+1))
		str.WriteString(line)
		str.WriteString("\n")
	}

	return str.String()
}

// AnyOverlapping returns true if any of the edits overlap. Edits are considered
// to overlap if the end of one edit is greater than or equal to the start of
// the next.
// 
// Note that this function mutates the input slice by sorting it by start line.
func AnyOverlapping(edits []Edit) bool {
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].Start < edits[j].Start
	})

	for i := 0; i < len(edits)-1; i++ {
		if edits[i].End > 0 && edits[i].End >= edits[i+1].Start {
			return true
		}
	}

	return false
}

func Apply(lines []string, edits []Edit) []string {
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].Start < edits[j].Start
	})

	result := make([]string, len(lines))
	copy(result, lines)
	offset := 0

	for _, e := range edits {
		start := e.Start + offset - 1
		end := start
		if e.End > 0 {
			// No -1 here because the end index is exclusive, but e.End is inclusive.
			end = e.End + offset
		}

		delta := len(e.Replacement) - (end - start)
		updated := make([]string, 0, len(result)+delta)
		updated = append(updated, result[:start]...)
		updated = append(updated, e.Replacement...)
		updated = append(updated, result[end:]...)
		offset += delta
		result = updated
	}

	return result
}
