#!/bin/bash
# Create malicious tar files for testing security

echo "Creating malicious test archives..."

# 1. Path traversal attack
mkdir -p malicious1
echo "malicious content" > malicious1/../../etc/passwd
tar -cf path-traversal.tar -C malicious1 .
rm -rf malicious1

# 2. Symlink attack  
mkdir -p malicious2
ln -s /etc/passwd malicious2/symlink-attack
tar -cf symlink-attack.tar -C malicious2 .
rm -rf malicious2

# 3. Large file (zip bomb simulation)
mkdir -p malicious3
dd if=/dev/zero of=malicious3/largefile bs=1M count=200 2>/dev/null
tar -cf large-file.tar -C malicious3 .
rm -rf malicious3

echo "Created malicious test files:"
echo "- path-traversal.tar"
echo "- symlink-attack.tar" 
echo "- large-file.tar"