// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"log"
	"os"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerlib"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/exe"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("imagecustomizer", "Customizes a pre-built CBL-Mariner image")

	buildDir          = app.Flag("build-dir", "Directory to run build out of.").Required().String()
	imageFile         = app.Flag("image-file", "Path of the base CBL-Mariner image which the customization will be applied to.").Required().String()
	outputImageFile   = app.Flag("output-image-file", "Path to write the customized image to.").Required().String()
	outputImageFormat = app.Flag("output-image-format", "Format of output image. Supported: vhd, vhdx, qcow2, raw.").Required().Enum("vhd", "vhdx", "qcow2", "raw")
	configFile        = app.Flag("config-file", "Path of the image customization config file.").Required().String()
	rpmSources        = app.Flag("rpm-source", "Path of the image customization config file.").Strings()
	logFile           = exe.LogFileFlag(app)
	logLevel          = exe.LogLevelFlag(app)
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	logger.InitBestEffort(*logFile, *logLevel)

	err := customizeImage()
	if err != nil {
		log.Fatalf("image customization failed: %v", err)
	}
}

func customizeImage() error {
	var err error

	err = imagecustomizerlib.CustomizeImageWithConfigFile(*buildDir, *configFile, *imageFile,
		rpmSources, *outputImageFile, *outputImageFormat)
	if err != nil {
		return err
	}

	return nil
}
