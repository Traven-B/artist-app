#!/bin/sh
# Hardcoded SCSS source and target
# Assumes main.scss exists in scss/ and static/ is target

SASS_CMD="/home/b/bin/sass"      # your dart-sass absolute path
SRC="/home/b/work/go/artistapp/scss/main.scss"
TARGET="/home/b/work/go/artistapp/static/main.css"

echo "Building SCSS..."
$SASS_CMD "$SRC" "$TARGET"
RET=$?

if [ $RET -eq 0 ]; then
    echo "SCSS build succeeded -> $TARGET"
else
    echo "SCSS build failed!"
fi

exit $RET
