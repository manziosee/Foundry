#!/bin/bash
# Create a test image tarball for testing

mkdir -p test-rootfs/bin test-rootfs/etc
echo "#!/bin/sh" > test-rootfs/bin/hello
echo "echo 'Hello from container!'" >> test-rootfs/bin/hello
chmod +x test-rootfs/bin/hello
echo "test:x:1000:1000:test:/:/bin/sh" > test-rootfs/etc/passwd

tar -cf test-image.tar -C test-rootfs .
sha256sum test-image.tar

rm -rf test-rootfs
echo "Created test-image.tar"