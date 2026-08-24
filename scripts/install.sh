#!/bin/sh
set -e

# install.sh — build and install the canon binary.
#
# Usage:
#   scripts/install.sh
#   scripts/install.sh --prefix /usr/local
#   scripts/install.sh --bindir "$HOME/bin"
#   scripts/install.sh --dry-run
#   scripts/install.sh --uninstall

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
BINARY_NAME="canon"

if [ "$(id -u)" -eq 0 ]; then
    DEFAULT_PREFIX="/usr/local"
else
    DEFAULT_PREFIX="${HOME}/.local"
fi

PREFIX="$DEFAULT_PREFIX"
BINDIR=""
DRY_RUN=0
UNINSTALL=0

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --prefix DIR   Installation prefix (default: $DEFAULT_PREFIX)
  --bindir DIR   Directory for the binary (default: PREFIX/bin)
  --dry-run      Print the install plan without making changes
  --uninstall    Remove the installed binary
  -h, --help     Show this help message
EOF
    exit 1
}

while [ $# -gt 0 ]; do
    case "$1" in
        --prefix)
            [ -n "$2" ] || usage
            PREFIX="$2"
            shift 2
            ;;
        --bindir)
            [ -n "$2" ] || usage
            BINDIR="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        --uninstall)
            UNINSTALL=1
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage
            ;;
    esac
done

if [ -z "$BINDIR" ]; then
    BINDIR="$PREFIX/bin"
fi

TARGET="$BINDIR/$BINARY_NAME"

if [ "$UNINSTALL" -eq 1 ]; then
    if [ -e "$TARGET" ]; then
        if [ "$DRY_RUN" -eq 1 ]; then
            echo "Would remove $TARGET"
        else
            rm -f "$TARGET"
            echo "Removed $TARGET"
        fi
    else
        echo "$TARGET is not installed" >&2
        exit 1
    fi
    exit 0
fi

if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is required to build canon but was not found in PATH" >&2
    exit 1
fi

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

cd "$REPO_ROOT"
VERSION=$(git describe --tags --long --always --dirty 2>/dev/null || echo dev)
go build -ldflags "-X github.com/victorhsb/canon/internal/canon.Version=$VERSION" -o "$BUILD_DIR/$BINARY_NAME" ./cmd/canon

if [ "$DRY_RUN" -eq 1 ]; then
    echo "Would install $TARGET"
    echo "Would run: mkdir -p $BINDIR"
    echo "Would run: cp $BUILD_DIR/$BINARY_NAME $TARGET"
    echo "Would run: chmod 755 $TARGET"
    exit 0
fi

mkdir -p "$BINDIR"
cp "$BUILD_DIR/$BINARY_NAME" "$TARGET"
chmod 755 "$TARGET"

echo "Installed $TARGET"
"$TARGET" commands >/dev/null
echo "Verified installation"
