// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package yamlutils

import (
	"io/ioutil"
	"os"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"gopkg.in/yaml.v3"
)

const (
	defaultYamlFilePermission os.FileMode = 0664
)

// ReadYAMLFile reads a YAML file.
func ReadYAMLFile(path string, data interface{}) error {
	yamlFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer yamlFile.Close()

	yamlData, err := ioutil.ReadAll(yamlFile)
	if err != nil {
		return err
	}

	logger.Log.Tracef("Read %#x bytes of YAML data.", len(yamlData))

	return yaml.Unmarshal(yamlData, data)
}

// WriteYAMLFile writes a yaml file.
func WriteYAMLFile(outputFilePath string, data interface{}) error {
	outputBytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	logger.Log.Tracef("Writing %#x bytes of YAML data.", len(outputBytes))

	return ioutil.WriteFile(outputFilePath, outputBytes, defaultYamlFilePermission)
}
