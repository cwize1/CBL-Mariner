// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// Library for parsing the RPM .spec file package dependency format.

package main

import (
	"fmt"
	"regexp"
)

type DependencyExprType int

const (
	PackageExprType DependencyExprType = iota
	OrExprType
	AndExprType
	IfExprType
	WithExprType
	WithoutExprType
	UnlessExprType
)

var (
	packageExprRegex    = regexp.MustCompile(`^([a-zA-Z0-9\-\._]+)( +(<=|>=|<|>|=) +([0-9\.]+))? *`)
	booleanKeywordRegex = regexp.MustCompile(`^(and|or|if|unless|without|with) *`)
	elseKeywordRegex    = regexp.MustCompile(`^else *`)
	andKeywordRegex     = regexp.MustCompile(`^and *`)
	orKeywordRegex      = regexp.MustCompile(`^or *`)
	closeParenRegex     = regexp.MustCompile(`^\) *`)
)

type DependencyExpr struct {
	Type    DependencyExprType
	Package *PackageExpr
	Clauses []*DependencyExpr
}

type PackageExpr struct {
	Name              string
	VersionComparison string
	Version           string
}

func ParseDependencyExpr(packageString string) (*DependencyExpr, error) {
	expr, remainder, err := parseDependencyExpr(packageString)
	if err != nil {
		index := len(packageString) - len(remainder)
		return nil, fmt.Errorf("dependency expression parse error at index %d: %w", index, err)
	}

	if len(remainder) != 0 {
		return nil, fmt.Errorf("expecting end of string")
	}

	return expr, nil
}

func parseDependencyExpr(packageString string) (*DependencyExpr, string, error) {
	if packageString[0] == '(' {
		return parseNestedExpr(packageString)
	} else {
		return parsePackageExpr(packageString)
	}
}

func parseNestedExpr(packageString string) (*DependencyExpr, string, error) {
	var err error

	// Consume the '(' char.
	packageString = packageString[1:]

	var firstExpr *DependencyExpr
	firstExpr, packageString, err = parseDependencyExpr(packageString)
	if err != nil {
		return nil, packageString, err
	}

	keywordMatch := booleanKeywordRegex.FindStringSubmatch(packageString)
	if keywordMatch == nil {
		return nil, packageString, fmt.Errorf("invalid boolean expression")
	}

	// Consume the regex match.
	packageString = packageString[len(keywordMatch[0]):]

	keyword := keywordMatch[1]

	var result *DependencyExpr
	switch keyword {
	case "and":
		result, packageString, err = parseBooleanChainExpr(firstExpr, packageString, andKeywordRegex, AndExprType)

	case "or":
		result, packageString, err = parseBooleanChainExpr(firstExpr, packageString, orKeywordRegex, OrExprType)

	case "if":
		result, packageString, err = parseIfExpr(firstExpr, packageString, IfExprType)

	case "unless":
		result, packageString, err = parseIfExpr(firstExpr, packageString, UnlessExprType)

	case "with":
		result, packageString, err = parseWithExpr(firstExpr, packageString, WithExprType)

	case "without":
		result, packageString, err = parseWithExpr(firstExpr, packageString, WithoutExprType)

	default:
		return nil, packageString, fmt.Errorf("expecting boolean keyword ('or', 'and', 'if', 'without', or 'unless')")
	}

	if err != nil {
		return nil, packageString, err
	}

	closeParenMatch := closeParenRegex.FindString(packageString)
	if closeParenMatch == "" {
		return nil, packageString, fmt.Errorf("expecting ')' character")
	}

	// Consume the regex match.
	packageString = packageString[len(closeParenMatch):]

	return result, packageString, nil
}

func parsePackageExpr(packageString string) (*DependencyExpr, string, error) {
	match := packageExprRegex.FindStringSubmatch(packageString)
	if match == nil {
		return nil, packageString, fmt.Errorf("invalid package expression")
	}

	// Consume the regex match.
	packageString = packageString[len(match[0]):]

	result := &DependencyExpr{
		Type: PackageExprType,
		Package: &PackageExpr{
			Name:              match[1],
			VersionComparison: match[3],
			Version:           match[4],
		},
	}

	return result, packageString, nil
}

func parseBooleanChainExpr(firstExpr *DependencyExpr, packageString string, keywordRegex *regexp.Regexp, exprType DependencyExprType) (*DependencyExpr, string, error) {
	var err error

	clauses := []*DependencyExpr{firstExpr}

	for {
		// Parse next expression.
		var nextExpr *DependencyExpr
		nextExpr, packageString, err = parseDependencyExpr(packageString)
		if err != nil {
			return nil, packageString, err
		}

		clauses = append(clauses, nextExpr)

		// Look for and chained 'or' or 'and' keyword.
		match := keywordRegex.FindString(packageString)
		if match == "" {
			break
		}

		// Consume the regex match.
		packageString = packageString[len(match):]
		continue
	}

	result := &DependencyExpr{
		Type:    exprType,
		Clauses: clauses,
	}

	return result, packageString, nil
}

func parseIfExpr(firstExpr *DependencyExpr, packageString string, exprType DependencyExprType) (*DependencyExpr, string, error) {
	var err error

	clauses := []*DependencyExpr{firstExpr}

	var nextExpr *DependencyExpr
	nextExpr, packageString, err = parseDependencyExpr(packageString)
	if err != nil {
		return nil, packageString, err
	}

	clauses = append(clauses, nextExpr)

	// Check for the 'else' keyword.
	match := elseKeywordRegex.FindString(packageString)
	if match != "" {
		// Consume the 'else ' chars.
		packageString = packageString[len(match):]

		// Parse next expression.
		nextExpr, packageString, err = parseDependencyExpr(packageString)
		if err != nil {
			return nil, packageString, err
		}

		clauses = append(clauses, nextExpr)
	}

	result := &DependencyExpr{
		Type:    exprType,
		Clauses: clauses,
	}

	return result, packageString, nil
}

func parseWithExpr(firstExpr *DependencyExpr, packageString string, exprType DependencyExprType) (*DependencyExpr, string, error) {
	var err error

	clauses := []*DependencyExpr{firstExpr}

	var nextExpr *DependencyExpr
	nextExpr, packageString, err = parseDependencyExpr(packageString)
	if err != nil {
		return nil, packageString, err
	}

	clauses = append(clauses, nextExpr)

	result := &DependencyExpr{
		Type:    exprType,
		Clauses: clauses,
	}

	return result, packageString, nil
}
