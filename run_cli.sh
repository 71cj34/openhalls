#!/bin/sh

SEARCH_DIR="$PWD"
NAME="hallview"

TMP="$(mktemp 2>/dev/null || echo /tmp/run_cli.$$)"
trap 'rm -f "$TMP"' EXIT INT TERM

find "$SEARCH_DIR" -type f -name "$NAME" -print 2>/dev/null | sort > "$TMP"
COUNT=$(wc -l < "$TMP" | tr -d '[:space:]')
case "$COUNT" in
    ''|*[!0-9]*) COUNT=0 ;;
esac

if [ "$COUNT" -eq 0 ]; then
    echo "No $NAME found under \"$SEARCH_DIR\"."
    # Helpful hint for trying to use Windows on Linux :D
    if find "$SEARCH_DIR" -type f -name 'hallview.exe' -print 2>/dev/null | grep -q .; then
        echo "A Windows hallview.exe was found, but it cannot run natively on Linux/macOS."
        echo "Build the native binary with: ./src/hallview/buildonly.sh"
    else
        echo "Build it first with ./src/hallview/buildonly.sh, or download it from the releases page."
    fi
    printf 'Press ENTER to exit... '
    IFS= read -r _ || true
    exit 1
fi

if [ "$COUNT" -gt 1 ]; then
    echo "Found $COUNT copies of $NAME:"
    sed 's/^/  /' "$TMP"
    echo ""
    echo "Please delete or move all but one of them, then run this script again."
    echo ""
    printf 'Press ENTER to exit... '
    IFS= read -r _ || true
    exit 1
fi

FOUND="$(cat "$TMP")"
echo "Found: \"$FOUND\""
echo ""

while true; do
    printf 'Press ENTER to run it, n to exit: '
    if ! IFS= read -r CHOICE; then
        echo ""
        exit 0
    fi
    case "$CHOICE" in
        '')
            echo "Running \"$FOUND\" ..."
            exec "$FOUND" "$@"
            ;;
        [nN])
            exit 0
            ;;
        *)
            echo "Please press ENTER to run, or type n to exit."
            ;;
    esac
done
