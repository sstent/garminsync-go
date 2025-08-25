#!/bin/bash
# Move activity.go back to parser directory
mv internal/parser/activity.go internal/parser/activity.go

# Remove extra directories
rm -rf internal/parser/activity

# Revert import paths
find . -name '*.go' -exec sed -i \
  -e 's|parser/activity/activity|parser|g' \
  -e 's|github.com/sstent/garminsync-go/internal/parser/activity/activity|github.com/sstent/garminsync-go/internal/parser|g' \
  {} +

# Remove unused fix_imports script
rm -f fix_imports.sh
