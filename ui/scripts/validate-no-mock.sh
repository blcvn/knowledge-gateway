#!/bin/bash
# ui/scripts/validate-no-mock.sh
# TASK-API-014: Validate no mock references remain in source code

set -e
FAILED=0

echo "================================================================"
echo " VNP Memory UI — Mock Cleanup Validation"
echo "================================================================"
echo ""

echo "Checking for mock imports in src/..."
MOCK_IMPORTS=$(grep -r "from.*mock\|import.*mock" \
  /Users/binhnt/Work/blockchain/vnp-memory/ui/src \
  --include="*.ts" --include="*.tsx" \
  | grep -v "node_modules" \
  | grep -v "\.mock\.ts:" \
  | grep -v "mock/index.ts" \
  | wc -l | tr -d ' ')

if [ "$MOCK_IMPORTS" -gt 0 ]; then
  echo "❌ Found $MOCK_IMPORTS mock imports:"
  grep -r "from.*mock\|import.*mock" \
    /Users/binhnt/Work/blockchain/vnp-memory/ui/src \
    --include="*.ts" --include="*.tsx" \
    | grep -v "node_modules" | grep -v "\.mock\.ts:" | grep -v "mock/index.ts"
  FAILED=1
else
  echo "✅ No mock imports found"
fi

echo ""
echo "Checking for useMockData / useMock flags..."
MOCK_FLAGS=$(grep -r "useMockData\|useMock " \
  /Users/binhnt/Work/blockchain/vnp-memory/ui/src \
  --include="*.ts" --include="*.tsx" \
  | grep -v "node_modules" | wc -l | tr -d ' ')

if [ "$MOCK_FLAGS" -gt 0 ]; then
  echo "❌ Found $MOCK_FLAGS useMockData/useMock references:"
  grep -r "useMockData\|useMock " \
    /Users/binhnt/Work/blockchain/vnp-memory/ui/src \
    --include="*.ts" --include="*.tsx" | grep -v "node_modules"
  FAILED=1
else
  echo "✅ No useMockData flags found"
fi

echo ""
echo "Running TypeScript check..."
cd /Users/binhnt/Work/blockchain/vnp-memory/ui
if node_modules/.bin/tsc --noEmit 2>&1; then
  echo "✅ TypeScript OK"
else
  echo "❌ TypeScript errors found"
  FAILED=1
fi

echo ""
echo "Running vite build..."
if node_modules/.bin/vite build 2>&1 | tail -5; then
  echo "✅ Build OK"
else
  echo "❌ Build failed"
  FAILED=1
fi

echo ""
echo "================================================================"
if [ "$FAILED" -eq 0 ]; then
  echo " ✅ ALL CHECKS PASSED — Mock migration complete!"
else
  echo " ❌ SOME CHECKS FAILED — See details above"
  exit 1
fi
echo "================================================================"
