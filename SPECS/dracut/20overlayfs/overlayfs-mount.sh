#!/bin/bash
# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

# Description: This script is designed to mount a DM-Verity root filesystem and
# set up OverlayFS. It is driven by kernel parameters and is invoked during the
# dracut initramfs phase.

# Kernel Parameters:
# - root: Specifies the path to the root filesystem. This script is designed to
#   support both DM-Verity protected devices and general filesystems. When a
#   DM-Verity protected device is detected (typically '/dev/mapper/root' for
#   systemd), the script performs steps specific to Verity. For non-DM-Verity
#   setups, the script will proceed with the standard OverlayFS setup, ensuring
#   versatility in its application.
# - rd.overlayfs: A comma-separated list defining the OverlayFS configuration.
#   Each entry should specify the overlay, upper, work directories, and optional
#   volume for an OverlayFS instance.

# Behavior:
# - Verifies the presence of the 'dracut-lib' for necessary utilities.
# - Mounts the DM-Verity root filesystem as read-only at a predefined mount
#   point.
# - Sets up the OverlayFS based on the provided kernel parameters. If a
#   persistent volume is specified, it's used as the upper layer for the
#   OverlayFS; otherwise, a volatile overlay is created.
# - Mounts the OverlayFS on top of the root filesystem, merging the read-only
#   root with the writable overlay, allowing system modifications without
#   altering the base system.

parse_kernel_cmdline_args() {
    # Ensure that the 'dracut-lib' is present and loaded.
    type getarg >/dev/null 2>&1 || . /lib/dracut-lib.sh

    OVERLAY_MOUNT="/run/overlay_mnt"
    OVERLAY_MNT_OPTS="rw,nodev,nosuid,nouser,noexec"

    # Retrieve the OverlayFS parameters.
    [ -z "${overlayfs}" ] && overlayfs=$(getarg rd.overlayfs=)
}

# Modified function to mount volatile or persistent volume.
mount_volatile_persistent_volume() {
    local _volume=$1
    local _overlay_mount=$2

    mkdir -p "${_overlay_mount}"

    if [[ "${_volume}" == "volatile" ]]; then
        # Fallback to volatile overlay if no persistent volume is specified.
        echo "No overlayfs persistent volume specified. Creating a volatile overlay."
        mount -t tmpfs tmpfs -o ${OVERLAY_MNT_OPTS} "${_overlay_mount}" || \
            die "Failed to create overlay tmpfs at ${_overlay_mount}"
    else
        # Check if the specified Overlay RAID volume is present in the system.
        if mdadm --examine --scan | grep -qs "${_volume}"; then
            # If the specified Overlay RAID volume is found, attempt to assemble it.
            mdadm --assemble "${_volume}" || \
                die "Failed to assemble RAID volume."
        fi

        # Mount the specified persistent volume.
        mount "${_volume}" "${_overlay_mount}" || \
            die "Failed to mount ${_volume} at ${_overlay_mount}"
    fi
}

create_overlayfs() {
    local _lower="$1"
    local _upper="$2"
    local _work="$3"
    local _options="$4"
    
    if [[ "$_options" != "" ]]; then
        _options="${_options},"
    fi

    [ -d "$_lower" ] || die "Unable to create overlay as $_lower does not exist"

    mkdir -p "${_upper}" && \
    mkdir -p "${_work}" && \
    mount -t overlay overlay -o "${_options}lowerdir=${_lower},upperdir=${_upper},workdir=${_work}" "${_lower}" || \
        die "Failed to mount overlay in ${_lower}"
}

mount_overlayfs() {
    local cnt=0
    local overlay_mount_with_cnt
    declare -A volume_mount_map

    echo "Starting to create OverlayFS"
    for _group in ${overlayfs}; do
        IFS=',' read -r overlay upper work volume options <<< "$_group"

        # Resolve volume to its full device path.
        volume=$(expand_persistent_dev "$volume")

        if [[ "$volume" == "" ]]; then
            overlay_mount_with_cnt="${OVERLAY_MOUNT}/${cnt}"
            mount_volatile_persistent_volume "volatile" $overlay_mount_with_cnt
        else
            if [[ -n "${volume_mount_map[$volume]}" ]]; then
                # Volume already mounted, retrieve existing mount point from map.
                overlay_mount_with_cnt=${volume_mount_map[$volume]}
            else
                # Not in map, so mount and update the map.
                overlay_mount_with_cnt="${OVERLAY_MOUNT}/${cnt}"
                mount_volatile_persistent_volume $volume $overlay_mount_with_cnt
                volume_mount_map[$volume]=$overlay_mount_with_cnt
            fi
        fi
        cnt=$((cnt + 1))

        echo "Creating OverlayFS with overlay: $overlay, upper: ${overlay_mount_with_cnt}/${upper#/}, work: ${overlay_mount_with_cnt}/${work#/}, options:${options}"
        create_overlayfs "${NEWROOT}/${overlay#/}" "${overlay_mount_with_cnt}/${upper#/}" "${overlay_mount_with_cnt}/${work#/}"
    done

    echo "Done OverlayFS Mounting"
}

# Keep a copy of this function here from verity-read-only-root package.
expand_persistent_dev() {
    local _dev=$1

    case "$_dev" in
        LABEL=*)
            _dev="/dev/disk/by-label/${_dev#LABEL=}"
            ;;
        UUID=*)
            _dev="${_dev#UUID=}"
            _dev="${_dev,,}"
            _dev="/dev/disk/by-uuid/${_dev}"
            ;;
        PARTUUID=*)
            _dev="${_dev#PARTUUID=}"
            _dev="${_dev,,}"
            _dev="/dev/disk/by-partuuid/${_dev}"
            ;;
        PARTLABEL=*)
            _dev="/dev/disk/by-partlabel/${_dev#PARTLABEL=}"
            ;;
    esac
    printf "%s" "$_dev"
}

# Parse kernel command line arguments to set environment variables.
# This function populates variables based on the kernel command line, such as overlayfs.
parse_kernel_cmdline_args

# Check if the overlayfs variable is set, indicating that overlay filesystem parameters were found.
# If not set, the process to enable and mount the overlay filesystem will be skipped.
if [ -n "${overlayfs}" ]; then
    mount_overlayfs
else
    echo "OverlayFS parameter not found in kernel cmdline, skipping mount_overlayfs."
fi
