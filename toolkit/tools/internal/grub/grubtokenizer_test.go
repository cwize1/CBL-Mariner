// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package grub

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeGrubConfig(t *testing.T) {
	testsDir := "tokentests/tests"
	expectedDir := "tokentests/expected"
	actualDir := "tokentests/actual"

	testFiles, err := os.ReadDir(testsDir)
	if !assert.NoError(t, err) {
		return
	}

	for _, testFile := range testFiles {
		testName := testFile.Name()

		grubConfig, err := os.ReadFile(filepath.Join(testsDir, testFile.Name()))
		if !assert.NoErrorf(t, err, "[%s] Read test file", testName) {
			continue
		}

		tokens, err := TokenizeGrubConfig(string(grubConfig))
		actual := tokenGrubConfigResultString(tokens, err)

		err = os.WriteFile(filepath.Join(actualDir, testFile.Name()), []byte(actual), os.ModePerm)
		assert.NoErrorf(t, err, "[%s] Write actual file", testName)

		expected, err := os.ReadFile(filepath.Join(expectedDir, testFile.Name()))
		if assert.NoErrorf(t, err, "[%s] Read expected file", testName) {
			assert.Equal(t, string(expected), actual)
		}
	}
}

func tokenGrubConfigResultString(tokens []Token, err error) string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("Error:\n%v\n\n", err))
	sb.WriteString("Tokens:\n")

	for _, token := range tokens {
		sb.WriteString(fmt.Sprintf("%s[%d:%d-%d:%d][%d-%d]\n", tokenTypeString(token.Type),
			token.Loc.Start.Line, token.Loc.Start.Col, token.Loc.End.Line, token.Loc.End.Col,
			token.Loc.Start.Index, token.Loc.End.Index))

		for _, subWord := range token.SubWords {
			sb.WriteString(fmt.Sprintf("  %s[%d:%d-%d:%d][%d-%d](%s)\n", subWordTypeString(subWord.Type),
				subWord.Loc.Start.Line, subWord.Loc.Start.Col, subWord.Loc.End.Line, subWord.Loc.End.Col,
				subWord.Loc.Start.Index, subWord.Loc.End.Index, strconv.Quote(subWord.Value)))
		}
	}

	return sb.String()
}

func tokenTypeString(tokenType TokenType) string {
	switch tokenType {
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case BAR:
		return "BAR"
	case AND:
		return "AND"
	case SEMICOLON:
		return "SEMICOLON"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case SPACE:
		return "SPACE"
	case NEWLINE:
		return "NEWLINE"
	case WORD:
		return "WORD"
	case COMMENT:
		return "COMMENT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", tokenType)
	}
}

func subWordTypeString(subWordType SubWordType) string {
	switch subWordType {
	case KEYWORD_STRING:
		return "KEYWORD_STRING"
	case STRING:
		return "STRING"
	case VAR_EXPANSION:
		return "VAR_EXPANSION"
	case QUOTED_VAR_EXPANSION:
		return "QUOTED_VAR_EXPANSION"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", subWordType)
	}
}
