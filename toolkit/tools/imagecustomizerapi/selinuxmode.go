package imagecustomizerapi

import (
	"fmt"
)

// SELinux sets the SELinux mode
type SELinuxMode string

const (
	// SELinuxModeDefault keeps the base image's existing SELinux mode.
	SELinuxModeDefault SELinuxMode = ""
	// SELinuxModeDisabled disables SELinux
	SELinuxModeDisabled SELinuxMode = "disabled"
	// SELinuxModeEnforcing sets SELinux to enforcing
	SELinuxModeEnforcing SELinuxMode = "enforcing"
	// SELinuxModePermissive sets SELinux to permissive
	SELinuxModePermissive SELinuxMode = "permissive"
	// SELinuxModeForceEnforcing both sets SELinux to enforcing, and forces it via the kernel command line
	SELinuxModeForceEnforcing SELinuxMode = "force-enforcing"
)

func (s SELinuxMode) IsValid() error {
	switch s {
	case SELinuxModeDefault, SELinuxModeDisabled, SELinuxModeEnforcing, SELinuxModePermissive, SELinuxModeForceEnforcing:
		// All good.
		return nil

	default:
		return fmt.Errorf("invalid SELinux value (%v)", s)
	}
}
