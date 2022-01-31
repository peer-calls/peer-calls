#!/bin/bash

diff \
  <( \
    grep -o '"[A-Z_]\+"' server/configread.go | \
    grep -v 'PEERCALLS_' | sed 's/^/PEERCALLS_/; s/"//g' | sort | uniq \
  ) \
  <( \
    grep -o '`[A-Z_]\+`' README.md | \
    grep -v 'PEERCALLS_LOG' | sed 's/`//g' | sort | uniq \
  )

exit_code=$?

if [[ $exit_code -ne 0 ]]; then
  echo "Documented environment variables in README.md and those defined in"
  echo "server/configread.go do not match. Please fix the issue!"
fi

exit $exit_code
