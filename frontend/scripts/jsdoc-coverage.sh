#!/bin/bash

# JSDoc Coverage Report
# Counts documented vs undocumented functions/components in the codebase

THRESHOLD=${JSDOC_THRESHOLD:-70}

echo "JSDoc Coverage Report"
echo "========================"
echo ""

# Count all functions/components (React components start with capital letter)
total_components=$(grep -rE "^(export default|export )?(const|function) [A-Z]" src --include="*.js" | wc -l)
total_functions=$(grep -rE "^(export )?(const|function) [a-z]" src --include="*.js" | wc -l)
total=$((total_components + total_functions))

# Count JSDoc blocks — look for */ on the line immediately before a function/component declaration
jsdoc_blocks=$(grep -rB1 -E "^(export default|export )?(const|function)" src --include="*.js" | grep -c " \*/$")

# Calculate coverage
if [ $total -gt 0 ]; then
  coverage=$((jsdoc_blocks * 100 / total))
else
  coverage=0
fi

echo "Statistics:"
echo "  Total Components: $total_components"
echo "  Total Functions:  $total_functions"
echo "  Total Items:      $total"
echo "  Documented:       $jsdoc_blocks"
echo ""
echo "Coverage: ${coverage}% (threshold: ${THRESHOLD}%)"
echo ""

if [ "$coverage" -lt "$THRESHOLD" ]; then
  echo "FAIL: JSDoc coverage ${coverage}% is below threshold of ${THRESHOLD}%"
  exit 1
else
  echo "PASS: JSDoc coverage ${coverage}% meets threshold of ${THRESHOLD}%"
fi
