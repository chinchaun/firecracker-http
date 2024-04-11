#!/bin/bash
set -euo pipefail

# if the host is missing acl for the kvm access
# sudo apt-get -y install acl
# sudo su
# sudo setfacl -m u:${USER}:rw /dev/kvm

if ! [ -r /dev/kvm ] && ! [ -w /dev/kvm ]; then
    echo "cannot activate KVM for the user ${USER}"
    exit 1
fi

ARCH=$(uname -m)

echo "CNI Folders"
# CNI Plugins
sudo mkdir -p /opt/cni
if [ "$ARCH" = "aarch64" ]; then
    sudo cp -r $PWD/cni-plugins/1.3.0/aarch64/bin /opt/cni
else
    sudo cp -r $PWD/cni-plugins/1.3.0/x86_64/bin /opt/cni
fi


echo "CNI Network"
# Networks for firecracker
sudo mkdir -p /etc/cni/conf.d
sudo cp $PWD/open-fire.conflist /etc/cni/conf.d

echo "Firecracker Jailer"

if [ "$ARCH" = "aarch64" ]; then
    sudo ln -sfn $PWD/firecracker/release-v1.6.0-aarch64/firecracker-v1.6.0-aarch64 "/usr/bin/firecracker"
    sudo ln -sfn $PWD/firecracker/release-v1.6.0-aarch64/jailer-v1.6.0-aarch64 "/usr/bin/jailer"
    
else
    sudo ln -sfn $PWD/firecracker/release-v1.6.0-x86_64/firecracker-v1.6.0-x86_64 "/usr/bin/firecracker"
    sudo ln -sfn $PWD/firecracker/release-v1.6.0-x86_64/jailer-v1.6.0-x86_64 "/usr/bin/jailer"
fi


firecracker --help | head -n1
jailer --help | head -n1

echo "Jailer home"

jailer_home="${1:-false}"
if [ "$jailer_home"  = false ]
  then
    sudo mkdir -p /srv/jailer
    echo "/srv/jailer"
else
    sudo mkdir -p "$1"
    echo "$1"
fi

