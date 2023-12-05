// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/exe"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/timestamp"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/pkg/imagecustomizerlib"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/pkg/profile"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	ToolsBinName = "toolsbin.squashfs"
)

var (
	app = kingpin.New("imagecustomizer", "Customizes a pre-built CBL-Mariner image")

	buildDir                 = app.Flag("build-dir", "Directory to run build out of.").Required().String()
	imageFile                = app.Flag("image-file", "Path of the base CBL-Mariner image which the customization will be applied to.").Required().String()
	outputImageFile          = app.Flag("output-image-file", "Path to write the customized image to.").Required().String()
	outputImageFormat        = app.Flag("output-image-format", "Format of output image. Supported: vhd, vhdx, qcow2, raw.").Required().Enum("vhd", "vhdx", "qcow2", "raw")
	configFile               = app.Flag("config-file", "Path of the image customization config file.").Required().String()
	rpmSources               = app.Flag("rpm-source", "Path to a RPM repo config file or a directory containing RPMs.").Strings()
	disableBaseImageRpmRepos = app.Flag("disable-base-image-rpm-repos", "Disable the base image's RPM repos as an RPM source").Bool()
	toolsbin                 = app.Flag("tools-bin", "Manually specify the path of the toolsbin.squashfs file. Default directory is the exe's directory.").String()
	logFile                  = exe.LogFileFlag(app)
	logLevel                 = exe.LogLevelFlag(app)
	profFlags                = exe.SetupProfileFlags(app)
	timestampFile            = app.Flag("timestamp-file", "File that stores timestamps for this program.").String()
)

func main() {
	var err error

	app.Version(imagecustomizerlib.ToolVersion)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	logger.InitBestEffort(*logFile, *logLevel)

	prof, err := profile.StartProfiling(profFlags)
	if err != nil {
		logger.Log.Warnf("Could not start profiling: %s", err)
	}
	defer prof.StopProfiler()

	timestamp.BeginTiming("imagecustomizer", *timestampFile)
	defer timestamp.CompleteTiming()

	err = customizeImage()
	if err != nil {
		log.Fatalf("image customization failed: %v", err)
	}
}

func customizeImage() error {
	var err error

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	var toolsBinPath string
	if *toolsbin != "" {
		// User provided an explicit path for the toolsbin.sqaushfs file.
		toolsBinPath, err = filepath.Abs(*toolsbin)
		if err != nil {
			return err
		}
	} else {
		// Look for the toolsbin.squashfs file in the same directory as the executable.
		exeDirPath := filepath.Dir(exePath)
		toolsBinPath = filepath.Join(exeDirPath, ToolsBinName)

		toolsSquashfsExists, err := file.PathExists(toolsBinPath)
		if err != nil {
			return fmt.Errorf("failed to check if %s file exists:\n%w", ToolsBinName, err)
		}

		if !toolsSquashfsExists {
			logger.Log.Warnf("%s file is missing; will use host's tools", ToolsBinName)
			toolsBinPath = ""
		}
	}

	err = imagecustomizerlib.CustomizeImageWithConfigFile(*buildDir, *configFile, *imageFile,
		*rpmSources, *outputImageFile, *outputImageFormat, !*disableBaseImageRpmRepos, toolsBinPath)
	if err != nil {
		return err
	}

	return nil
}
