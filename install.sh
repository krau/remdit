#!/bin/bash
# This script automatically downloads the latest release of remdit from GitHub,
# determines the appropriate binary for your CPU architecture,
# and installs it to /usr/local/bin so it's available in your PATH.

API_URL="https://api.github.com/repos/krau/remdit/releases/latest"

echo "Fetching latest release info from $API_URL ..."
TAG=$(curl -s "$API_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
    echo "Error: Unable to fetch the latest release tag."
    exit 1
fi
echo "Latest release tag: $TAG"

ARCH=$(uname -m)
case "$ARCH" in
x86_64)
    ARCH="amd64"
    ;;
aarch64)
    ARCH="arm64"
    ;;
*)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac
echo "Detected architecture: $ARCH"

# Construct the download URL for the tarball using the tag and architecture.
# The file name format is: remdit-<tag>-linux-<arch>.tar.gz (e.g., remdit-v0.1.4-linux-arm64.tar.gz)
DOWNLOAD_URL="https://github.com/krau/remdit/releases/download/${TAG}/remdit-${TAG}-linux-${ARCH}.tar.gz"
echo "Download URL: $DOWNLOAD_URL"

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR" || {
    echo "Error: Could not change directory to $TMP_DIR"
    exit 1
}

echo "Downloading remdit tarball..."
if ! curl -L -o remdit.tar.gz "$DOWNLOAD_URL"; then
    echo "Error: Download failed."
    exit 1
fi

# Extract the downloaded tarball.
echo "Extracting remdit tarball..."
if ! tar -xzf remdit.tar.gz; then
    echo "Error: Extraction failed."
    exit 1
fi

# Verify that the remdit binary exists in the extracted contents.
if [ ! -f remdit ]; then
    echo "Error: remdit binary not found after extraction."
    exit 1
fi

# Ensure the remdit binary is executable.
chmod +x remdit

# Install the remdit binary to /usr/local/bin, which should be in the user's PATH.
# This step requires sudo privileges.
echo "Installing remdit to /usr/local/bin ..."
if ! sudo mv remdit /usr/local/bin/; then
    echo "Error: Installation failed."
    exit 1
fi

# Clean up by removing the temporary directory.
cd /
rm -rf "$TMP_DIR"

echo "remdit has been installed successfully and is available in your PATH."