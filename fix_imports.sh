#!/bin/bash
# Find all Go files and replace import paths
find . -name '*.go' -exec sed -i \
  -e 's|github.com/yourusername/garminsync|github.com/sstent/garminsync-go|g' \
  -e 's|github.com/sstent/garminsync-go/internal/parser/activity|github.com/sstent/garminsync-go/internal/parser/activity/activity|g' \
  {} +
