// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/stretchr/testify/assert"
)

var (
	tmpDir string
)

func TestMain(m *testing.M) {
	var err error

	logger.InitStderrLog()

	workingDir, err := os.Getwd()
	if err != nil {
		logger.Log.Panicf("Failed to get working directory, error: %s", err)
	}

	tmpDir = filepath.Join(workingDir, "_tmp")

	err = os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		logger.Log.Panicf("Failed to create temp directory, error: %s", err)
	}

	retVal := m.Run()

	err = os.RemoveAll(tmpDir)
	if err != nil {
		logger.Log.Warnf("Failed to cleanup tmp dir (%s). Error: %s", tmpDir, err)
	}

	os.Exit(retVal)
}

// testFileName returns a file name in a temporary directory. This path will
// be different for EVERY call to this function.
func testFileName(t *testing.T) string {
	return filepath.Join(t.TempDir(), t.Name())
}

func TestRemoveFileIfExistsValid(t *testing.T) {
	fileName := testFileName(t)
	// Create a file to remove
	err := Write("test", fileName)
	assert.NoError(t, err)

	err = RemoveFileIfExists(fileName)
	assert.NoError(t, err)

	exists, err := PathExists(fileName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRemoveFileDoesNotExist(t *testing.T) {
	fileName := testFileName(t)
	err := RemoveFileIfExists(fileName)
	assert.NoError(t, err)
}

func TestStringReplace(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("a{{.B}}cde"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = StringReplace("{{.B}}", "b", tmpFile)
	assert.NoError(t, err, "StringReplace failed")

	newContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err, "failed to read changed file")

	assert.Equal(t, "abcde", string(newContents))
}

func TestStringReplaceNop(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\na{{.B}}cde\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = StringReplace("{{.A}}", "b", tmpFile)
	assert.NoError(t, err, "StringReplace failed")

	newContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err, "failed to read changed file")

	assert.Equal(t, "first-line\na{{.B}}cde\nthird-line", string(newContents))
}

func TestStringRegexReplace(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\na=b\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = RegexpReplace("(?m)^(.*)=.*$", "${1}=z", tmpFile)
	assert.NoError(t, err, "RegexpReplace failed")

	newContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err, "failed to read changed file")

	assert.Equal(t, "first-line\na=z\nthird-line", string(newContents))
}

func TestStringRegexReplaceNop(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = RegexpReplace("(?m)^(.*)=.*$", "${1}=z", tmpFile)
	assert.NoError(t, err, "RegexpReplace failed")

	newContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err, "failed to read changed file")

	assert.Equal(t, "first-line\nthird-line", string(newContents))
}

func TestRegexpFindSubmatch(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\na=b\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	value, err := RegexpFindSubmatch("(?m)^a=(.*)$", 1, tmpFile)
	assert.NoError(t, err, "RegexpFindSubmatch failed")

	assert.Equal(t, "b", value)
}

func TestRegexpFindSubmatchMissing(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\na=b\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	_, err = RegexpFindSubmatch("(?m)^b=(.*)$", 1, tmpFile)
	assert.Error(t, err, "RegexpFindSubmatch should have failed")
	assert.ErrorContains(t, err, "match")
}

func TestInsertAtLine(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\nsecond-line\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = InsertAtLine(2, "inserted-line", tmpFile)
	assert.NoError(t, err, "InsertAtLine failed")

	newContents, err := os.ReadFile(tmpFile)
	assert.NoError(t, err, "failed to read changed file")

	assert.Equal(t, "first-line\ninserted-line\nsecond-line\nthird-line", string(newContents))
}

func TestInsertAtLineMissing(t *testing.T) {
	tmpFile := filepath.Join(tmpDir, "TestStringReplace")

	err := os.WriteFile(tmpFile, []byte("first-line\nsecond-line\nthird-line"), os.ModePerm)
	assert.NoError(t, err, "failed to write test file")

	err = InsertAtLine(10, "inserted-line", tmpFile)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "expected number of lines")
}
