#!/bin/bash
#
# test-validation.sh
# Validates the installation scripts without actually executing them
#

echo "========================================="
echo " ZID Proxy Installation Scripts Validation"
echo "========================================="
echo ""

SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR"

errors=0
warnings=0

# Test 1: Check all required files exist
echo "Test 1: Checking required files..."
files=(
    "install.sh"
    "activate-package.php"
    "register-package.php"
    "diagnose.sh"
    "uninstall.sh"
    "files/usr/local/pkg/zid-proxy.xml"
    "files/usr/local/pkg/zid-proxy.inc"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file"
    else
        echo "  ✗ $file (MISSING)"
        errors=$((errors + 1))
    fi
done
echo ""

# Test 2: Check shell script syntax
echo "Test 2: Validating shell script syntax..."
for script in install.sh diagnose.sh uninstall.sh; do
    if sh -n "$script" 2>/dev/null; then
        echo "  ✓ $script"
    else
        echo "  ✗ $script (SYNTAX ERROR)"
        errors=$((errors + 1))
    fi
done
echo ""

# Test 3: Check PHP script syntax
echo "Test 3: Validating PHP script syntax..."
for script in activate-package.php register-package.php; do
    if php -l "$script" > /dev/null 2>&1; then
        echo "  ✓ $script"
    else
        echo "  ✗ $script (SYNTAX ERROR)"
        errors=$((errors + 1))
    fi
done
echo ""

# Test 4: Check executable permissions
echo "Test 4: Checking executable permissions..."
for script in install.sh diagnose.sh uninstall.sh activate-package.php register-package.php; do
    if [ -x "$script" ]; then
        echo "  ✓ $script"
    else
        echo "  ⚠ $script (NOT EXECUTABLE)"
        warnings=$((warnings + 1))
    fi
done
echo ""

# Test 5: Verify paths in PHP scripts
echo "Test 5: Verifying paths in PHP scripts..."

# Check activate-package.php
if grep -q "'/usr/local/pkg/zid-proxy.inc'" activate-package.php; then
    echo "  ✓ activate-package.php: correct include path"
else
    echo "  ✗ activate-package.php: wrong include path"
    errors=$((errors + 1))
fi

if grep -q "'/usr/local/etc/rc.d/zid-proxy.sh'" activate-package.php; then
    echo "  ✓ activate-package.php: correct RC script path"
else
    echo "  ✗ activate-package.php: wrong RC script path"
    errors=$((errors + 1))
fi

# Check register-package.php
if grep -q "'/etc/inc/config.inc'" register-package.php; then
    echo "  ✓ register-package.php: correct config.inc path"
else
    echo "  ✗ register-package.php: wrong config.inc path"
    errors=$((errors + 1))
fi

echo ""

# Test 6: Check zid-proxy.inc RC file path
echo "Test 6: Checking zid-proxy.inc configuration..."
if grep -q "ZIDPROXY_RCFILE.*'/usr/local/etc/rc.d/zid-proxy.sh'" files/usr/local/pkg/zid-proxy.inc; then
    echo "  ✓ RC file path matches (zid-proxy.sh)"
else
    echo "  ✗ RC file path mismatch"
    errors=$((errors + 1))
fi
echo ""

# Test 7: Verify install.sh calls activation scripts
echo "Test 7: Checking install.sh integration..."
if grep -q "activate-package.php" install.sh; then
    echo "  ✓ install.sh calls activate-package.php"
else
    echo "  ⚠ install.sh doesn't call activate-package.php"
    warnings=$((warnings + 1))
fi

if grep -q "register-package.php" install.sh; then
    echo "  ✓ install.sh references register-package.php"
else
    echo "  ⚠ install.sh doesn't reference register-package.php"
    warnings=$((warnings + 1))
fi
echo ""

# Test 8: Check for common issues
echo "Test 8: Checking for common issues..."

# Check for hardcoded paths that might be wrong
if grep -r "/opt/" . --include="*.sh" --include="*.php" 2>/dev/null | grep -v test-validation; then
    echo "  ⚠ Found /opt/ paths (unusual for FreeBSD)"
    warnings=$((warnings + 1))
else
    echo "  ✓ No /opt/ paths found"
fi

# Check for linux-specific commands
if grep -r "systemctl\|systemd" . --include="*.sh" 2>/dev/null | grep -v test-validation; then
    echo "  ✗ Found systemd commands (wrong for FreeBSD)"
    errors=$((errors + 1))
else
    echo "  ✓ No systemd commands found"
fi

echo ""

# Summary
echo "========================================="
echo " Validation Results"
echo "========================================="
echo ""
echo "Errors: $errors"
echo "Warnings: $warnings"
echo ""

if [ $errors -eq 0 ]; then
    echo "✓ All critical tests passed!"
    echo ""
    echo "Scripts are ready to use on pfSense."
    echo ""
    echo "Next steps:"
    echo "1. Copy to pfSense: scp -r pkg-zid-proxy root@pfsense:/tmp/"
    echo "2. On pfSense: cd /tmp/pkg-zid-proxy && sh install.sh"
    exit 0
else
    echo "✗ Found $errors error(s) that must be fixed!"
    exit 1
fi
