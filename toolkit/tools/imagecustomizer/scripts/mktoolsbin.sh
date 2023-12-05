#!/usr/bin/env bash

# This script create a pseduo Mariner container that contains some of the tools required by the image customizer.
#
# The "container" is actually a squashfs filesystem. During customization, this filesystem will be mounted and will be
# used via chroot.
#
# In general, it is far easier to just include any required tools in the dependencies section of the README.md file.
# Tools are only included in the toolsbin if they are difficult to acquire or have known issues stemming from breaking
# changes between versions.

set -eux

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

ROOT_DIR=$(realpath "$SCRIPT_DIR/../../../..")
TOOLKIT_DIR="$ROOT_DIR/toolkit"

BUILD_DIR="$TOOLKIT_DIR/build/toolsbin"
TOOLS_DIR="$BUILD_DIR/rootfs"
REPOS_DIR="$BUILD_DIR/repos"

OUT_DIR="$TOOLKIT_DIR/out/tools"

SQUASH_FS_FILENAME="$OUT_DIR/toolsbin.squashfs"
SQUASH_FS_ALGORITHM=lz4

# Reset build directories.
rm -rf "$BUILD_DIR" || true

mkdir -p "$REPOS_DIR"
mkdir -p "$TOOLS_DIR"
mkdir -p "$OUT_DIR"

rm -f "$SQUASH_FS_FILENAME"

# Setup directory containing RPM repo files.
cp \
    "$ROOT_DIR/SPECS/mariner-repos/MICROSOFT-RPM-GPG-KEY" \
    "$ROOT_DIR/SPECS/mariner-repos/MICROSOFT-METADATA-GPG-KEY" \
    "$ROOT_DIR/SPECS/mariner-repos/mariner-official-base.repo" \
    "$REPOS_DIR"

sed -i "s|etc/pki/rpm-gpg/|$REPOS_DIR/|g" "$REPOS_DIR/mariner-official-base.repo"

# Install the tools required by the image customizer into the tools directory.
dnf \
    --installroot "$TOOLS_DIR" \
    "--setopt=reposdir=$REPOS_DIR" \
    --releasever "2.0" \
    -y \
    install \
        createrepo_c \

# Cleanup RPM junk.
dnf \
    --installroot "$TOOLS_DIR" \
    "--setopt=reposdir=$REPOS_DIR" \
    --releasever "2.0" \
    -y \
    clean all

# Precreate mount points
mkdir -p "$TOOLS_DIR/mnt/imageroot"

# Create squashfs file.
mksquashfs \
    "$TOOLS_DIR" \
    "$SQUASH_FS_FILENAME" \
    -comp "$SQUASH_FS_ALGORITHM"
