// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package userutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/randomization"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
)

const (
	ShadowFile        = "/etc/shadow"
	RootHomeDir       = "/root"
	UserHomeDirPrefix = "/home"
	RootUser          = "root"
)

func HashPassword(password string) (string, error) {
	const postfixLength = 12

	salt, err := randomization.RandomString(postfixLength, randomization.LegalCharactersAlphaNum)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt for hashed password: %w", err)
	}

	// Generate hashed password based on salt value provided.
	// -6 option indicates to use the SHA256/SHA512 algorithm
	stdout, _, err := shell.Execute("openssl", "passwd", "-6", "-salt", salt, password)
	if err != nil {
		return "", fmt.Errorf("failed to generate hashed password: %w", err)
	}

	hashedPassword := strings.TrimSpace(stdout)
	return hashedPassword, nil
}

func UserExists(username string, installChroot *safechroot.Chroot) (bool, error) {
	var userExists bool
	err := installChroot.UnsafeRun(func() error {
		_, stderr, err := shell.Execute("id", "-u", username)
		if err != nil {
			if !strings.Contains(stderr, "no such user") {
				return fmt.Errorf("failed to check if user exists (%s): %w", username, err)
			}

			userExists = false
		} else {
			userExists = true
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return userExists, nil
}

func UpdateUserPassword(username string, hashedPassword string, installChroot *safechroot.Chroot) error {
	const sedDelimiter = "|"

	findPattern := fmt.Sprintf("%v:x:", username)
	replacePattern := fmt.Sprintf("%v:%v:", username, hashedPassword)
	filePath := filepath.Join(installChroot.RootDir(), ShadowFile)

	err := sed(findPattern, replacePattern, sedDelimiter, filePath)
	if err != nil {
		return fmt.Errorf("failed to write (%s) hashed password to shadow file (%s): %w", username, filePath, err)
	}

	return nil
}

func AddUser(username string, hashedPassword string, uid string, installChroot *safechroot.Chroot) error {
	var args = []string{username, "-m", "-p", hashedPassword}
	if uid != "" {
		args = append(args, "-u", uid)
	}

	err := installChroot.UnsafeRun(func() error {
		return shell.ExecuteLive(false /*squashErrors*/, "useradd", args...)
	})
	if err != nil {
		return fmt.Errorf("failed to add user (%s): %w", username, err)
	}

	return nil
}

// chage works in the same way as invoking "chage -M passwordExpirationInDays username"
// i.e. it sets the maximum password expiration date.
func Chage(passwordExpirationInDays int64, username string) (err error) {
	var (
		shadow            []string
		usernameWithColon = fmt.Sprintf("%s:", username)
	)

	shadow, err = file.ReadLines(ShadowFile)
	if err != nil {
		return
	}

	for n, entry := range shadow {
		done := false
		// Entries in shadow are separated by colon and start with a username
		// Finding one that starts like that means we've found our entry
		if strings.HasPrefix(entry, usernameWithColon) {
			// Each line in shadow contains 9 fields separated by colon ("") in the following order:
			// login name, encrypted password, date of last password change,
			// minimum password age, maximum password age, password warning period,
			// password inactivity period, account expiration date, reserved field for future use
			const (
				passwordNeverExpiresValue = -1
				loginNameField            = 0
				encryptedPasswordField    = 1
				passwordChangedField      = 2
				minPasswordAgeField       = 3
				maxPasswordAgeField       = 4
				warnPeriodField           = 5
				inactivityPeriodField     = 6
				expirationField           = 7
				reservedField             = 8
				totalFieldsCount          = 9
			)

			fields := strings.Split(entry, ":")
			// Any value other than totalFieldsCount indicates error in parsing
			if len(fields) != totalFieldsCount {
				return fmt.Errorf("invalid shadow entry (%v) for user (%s): %d fields expected, but %d found", fields, username, totalFieldsCount, len(fields))
			}

			if passwordExpirationInDays == passwordNeverExpiresValue {
				// If passwordExpirationInDays is equal to -1, it means that password never expires.
				// This is expressed by leaving account expiration date field (and fields after it) empty.
				for _, fieldToChange := range []int{maxPasswordAgeField, warnPeriodField, inactivityPeriodField, expirationField, reservedField} {
					fields[fieldToChange] = ""
				}
				// Each user appears only once, since we found one, we are finished; save the changes and exit.
				done = true
			} else if passwordExpirationInDays < passwordNeverExpiresValue {
				// Values smaller than -1 make no sense
				return fmt.Errorf("invalid value for maximum user's (%s) password expiration: %d; should be greater than %d", username, passwordExpirationInDays, passwordNeverExpiresValue)
			} else {
				// If passwordExpirationInDays has any other value, it's the maximum expiration date: set it accordingly
				// To do so, we need to ensure that passwordChangedField holds a valid value and then sum it with passwordExpirationInDays.
				var (
					passwordAge     int64
					passwordChanged = fields[passwordChangedField]
				)

				if passwordChanged == "" {
					// Set to the number of days since epoch
					fields[passwordChangedField] = fmt.Sprintf("%d", int64(time.Since(time.Unix(0, 0)).Hours()/24))
				}
				passwordAge, err = strconv.ParseInt(fields[passwordChangedField], 10, 64)
				if err != nil {
					return
				}
				fields[expirationField] = fmt.Sprintf("%d", passwordAge+passwordExpirationInDays)

				// Each user appears only once, since we found one, we are finished; save the changes and exit.
				done = true
			}
			if done {
				// Create and save new shadow file including potential changes from above.
				shadow[n] = strings.Join(fields, ":")
				err = file.Write(strings.Join(shadow, "\n"), ShadowFile)
				return
			}
		}
	}

	return fmt.Errorf(`user "%s" not found when trying to change the password expiration date`, username)
}

func ConfigureUserGroupMembership(username string, primaryGroup string,
	secondaryGroups []string, installChroot *safechroot.Chroot,
) error {
	var err error

	// Update primary group
	if primaryGroup != "" {
		err = installChroot.UnsafeRun(func() error {
			return shell.ExecuteLive(false /*squashErrors*/, "usermod", "-g", primaryGroup, username)
		})
		if err != nil {
			return err
		}
	}

	// Update secondary groups
	if len(secondaryGroups) != 0 {
		allGroups := strings.Join(secondaryGroups, ",")
		err = installChroot.UnsafeRun(func() error {
			return shell.ExecuteLive(false /*squashErrors*/, "usermod", "-a", "-G", allGroups, username)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func ProvisionUserSSHCerts(username string, sshPubKeyPaths []string, installChroot *safechroot.Chroot) (err error) {
	var (
		pubKeyData []string
		exists     bool
	)
	const squashErrors = false
	const authorizedKeysTempFilePerms = 0644
	const authorizedKeysTempFile = "/tmp/authorized_keys"
	const sshDirectoryPermission = "0700"

	// Skip user SSH directory generation when not provided with public keys
	// Let SSH handle the creation of this folder on its first use
	if len(sshPubKeyPaths) == 0 {
		return
	}

	homeDir := UserHomeDirectory(username)
	userSSHKeyDir := filepath.Join(homeDir, ".ssh")
	authorizedKeysFile := filepath.Join(userSSHKeyDir, "authorized_keys")

	exists, err = file.PathExists(authorizedKeysTempFile)
	if err != nil {
		logger.Log.Warnf("Error accessing %s file : %v", authorizedKeysTempFile, err)
		return
	}
	if !exists {
		logger.Log.Debugf("File %s does not exist. Creating file...", authorizedKeysTempFile)
		err = file.Create(authorizedKeysTempFile, authorizedKeysTempFilePerms)
		if err != nil {
			logger.Log.Warnf("Failed to create %s file : %v", authorizedKeysTempFile, err)
			return
		}
	} else {
		err = os.Truncate(authorizedKeysTempFile, 0)
		if err != nil {
			logger.Log.Warnf("Failed to truncate %s file : %v", authorizedKeysTempFile, err)
			return
		}
	}
	defer os.Remove(authorizedKeysTempFile)

	for _, pubKey := range sshPubKeyPaths {
		logger.Log.Infof("Adding ssh key (%s) to user (%s)", filepath.Base(pubKey), username)
		relativeDst := filepath.Join(userSSHKeyDir, filepath.Base(pubKey))

		fileToCopy := safechroot.FileToCopy{
			Src:  pubKey,
			Dest: relativeDst,
		}

		err = installChroot.AddFiles(fileToCopy)
		if err != nil {
			return
		}

		logger.Log.Infof("Adding ssh key (%s) to user (%s) .ssh/authorized_users", filepath.Base(pubKey), username)
		pubKeyData, err = file.ReadLines(pubKey)
		if err != nil {
			logger.Log.Warnf("Failed to read from SSHPubKey : %v", err)
			return
		}

		// Append to the tmp/authorized_users file
		for _, sshkey := range pubKeyData {
			sshkey += "\n"
			err = file.Append(sshkey, authorizedKeysTempFile)
			if err != nil {
				logger.Log.Warnf("Failed to append to %s : %v", authorizedKeysTempFile, err)
				return
			}
		}
	}

	fileToCopy := safechroot.FileToCopy{
		Src:  authorizedKeysTempFile,
		Dest: authorizedKeysFile,
	}

	err = installChroot.AddFiles(fileToCopy)
	if err != nil {
		return
	}

	// Change ownership of the folder to belong to the user and their primary group
	err = installChroot.UnsafeRun(func() (err error) {
		// Find the primary group of the user
		stdout, stderr, err := shell.Execute("id", "-g", username)
		if err != nil {
			logger.Log.Warnf(stderr)
			return
		}

		primaryGroup := strings.TrimSpace(stdout)
		logger.Log.Debugf("Primary group for user (%s) is (%s)", username, primaryGroup)

		ownership := fmt.Sprintf("%s:%s", username, primaryGroup)
		err = shell.ExecuteLive(squashErrors, "chown", "-R", ownership, userSSHKeyDir)
		if err != nil {
			return
		}

		err = shell.ExecuteLive(squashErrors, "chmod", "-R", sshDirectoryPermission, userSSHKeyDir)
		return
	})

	if err != nil {
		return
	}

	return
}

func UserHomeDirectory(username string) string {
	if username == RootUser {
		return RootHomeDir
	} else {
		return filepath.Join(UserHomeDirPrefix, username)
	}
}

func ConfigureUserStartupCommand(username string, startupCommand string, installChroot *safechroot.Chroot) error {
	const (
		passwdFilePath = "etc/passwd"
		sedDelimiter   = "|"
	)

	if startupCommand == "" {
		return nil
	}

	logger.Log.Debugf("Updating user '%s' startup command to '%s'.", username, startupCommand)

	findPattern := fmt.Sprintf(`^\(%s.*\):[^:]*$`, username)
	replacePattern := fmt.Sprintf(`\1:%s`, startupCommand)
	filePath := filepath.Join(installChroot.RootDir(), passwdFilePath)
	err := sed(findPattern, replacePattern, sedDelimiter, filePath)
	if err != nil {
		return fmt.Errorf("failed to update user's (%s) startup command (%s): %w", username, startupCommand, err)
	}

	return nil
}

func sed(find, replace, delimiter, file string) (err error) {
	replacement := fmt.Sprintf("s%s%s%s%s%s", delimiter, find, delimiter, replace, delimiter)
	return shell.ExecuteLive(false /*squashErrors*/, "sed", "-i", replacement, file)
}
