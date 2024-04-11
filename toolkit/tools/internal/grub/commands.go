// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package grub

type Command struct {
	Name string
	Args []Token
}

func SplitTokensIntoCommands(tokens []Token) []Command {
	commands := []Command(nil)

	for i := 0; i < len(tokens); {

	}

	return commands
}
