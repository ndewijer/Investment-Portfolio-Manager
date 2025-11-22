#!/bin/bash

# JSDoc Coverage Report
# Counts documented vs undocumented functions/components in the codebase

echo "🔍 JSDoc Coverage Report"
echo "========================"
echo ""

# Count all functions/components (React components start with capital letter)
total_components=$(grep -rE "^(export default|export )?(const|function) [A-Z]" src --include="*.js" | wc -l)
total_functions=$(grep -rE "^(export )?(const|function) [a-z]" src --include="*.js" | wc -l)
total=$((total_components + total_functions))

# Count JSDoc blocks (/** ... */)
jsdoc_blocks=$(grep -rB1 "^(export default|export )?(const|function)" src --include="*.js" | grep -c "/\*\*")

# Calculate coverage
if [ $total -gt 0 ]; then
  coverage=$((jsdoc_blocks * 100 / total))
else
  coverage=0
fi

echo "📊 Statistics:"
echo "  Total Components: $total_components"
echo "  Total Functions:  $total_functions"
echo "  Total Items:      $total"
echo "  Documented:       $jsdoc_blocks"
echo ""
echo "📈 Coverage: ${coverage}%"
echo ""

# Run eslint to show missing JSDoc warnings
echo "⚠️  ESLint JSDoc Warnings:"
echo "========================"
npm run lint --silent 2>&1 | grep -E "(jsdoc/require-jsdoc|Missing JSDoc)" | head -20

echo ""
echo "💡 To see all JSDoc warnings: npm run lint"
echo "💡 To enforce JSDoc coverage: npm run lint:jsdoc"
