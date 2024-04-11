# Firecracker-http
This is a simple HTTP server that simplifies the quick start of using firecracker, you will have a VM with an IP, PID and access to the internet inside it.

```
sudo $(which go) run .
starting server
server listening on 8080 
```
## Start a VM
```
curl --location 'http://localhost:8080/create' \
--header 'Content-Type: application/json' \
--data '{
    "kernelPath": "/path-to/kernels/vmlinux-5.10-x86_64.bin",
    "rootDrivePath": "/path-to/filesystems/ubuntu-22.04.ext4",
    "cniNetworkName": "open-fire",
    "VcpuCount": 1,
    "MemSizeMib": 512,
    "debug": false,
    "enableSmt": false,
    "jailerChrootBase": "/home/srv/jailer",
    "metadata": {
        "data": "some data"
    }
}'

Response:
{
    "ip": "192.168.127.207",
    "pid": 28062,
    "vmId": "p8q1uadgmdx5a9lm59ci"
}

SSH into the machine

ssh -i ./ubuntu-22.04.id_rsa root@192.168.127.207
```

## Stop a VM

```
curl --location 'http://http://localhost:8080/stop' \
--header 'Content-Type: application/json' \
--data '{
    "vmmId": "p8q1uadgmdx5a9lm59ci",
    "pid": 28062,
    "arch": "x86_64",
    "jailerChrootBase": "/home/srv/jailer"
}'

Response:

VM with id: p8q1uadgmdx5a9lm59ci has been stopped

```
# Get Started

Clone this repo!

## Run setup.sh script
This script basically will handle the setup of the paths for the network config, cni, jailer and firecracker binaries.

```
cd host_setup
bash setup.sh
```

## Kernels
Inside the kernels folder, you have two kernels compiled with the networking functionality enabled.

## RootFS download them from  the firecracker S3

```
ARCH="$(uname -m)"

# Download a rootfs
wget https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.8/${ARCH}/ubuntu-22.04.ext4

# Download the ssh key for the rootfs
wget https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.8/${ARCH}/ubuntu-22.04.id_rsa

# Set user read permission on the ssh key
chmod 400 ./ubuntu-22.04.id_rsa
```

## Install Go

https://www.digitalocean.com/community/tutorials/how-to-install-go-on-ubuntu-20-04

### x86_64
```
curl -OL https://golang.org/dl/go1.20.12.linux-amd64.tar.gz
```
### ARM
```
curl -OL https://go.dev/dl/go1.20.12.linux-arm64.tar.gz
```

## Run the server

```
sudo $(which go) run .

OR

sudo PORT=8081 $(which go) run .

```
The server needs to run with sudo because the jailer and the firecracker will need the sudo permission to run

## Help & Issues

### VM is not reaching internet

If you installed docker the nameserver may be get screw and also on reboot the Ipam plugin overrides the /etc/resolv.conf.

Solution

```
cat > /etc/systemd/resolved.conf <<'EOF'
#  This file is part of systemd.
#
#  systemd is free software; you can redistribute it and/or modify it under the
#  terms of the GNU Lesser General Public License as published by the Free
#  Software Foundation; either version 2.1 of the License, or (at your option)
#  any later version.
#
# Entries in this file show the compile time defaults. Local configuration
# should be created by either modifying this file, or by creating "drop-ins" in
# the resolved.conf.d/ subdirectory. The latter is generally recommended.
# Defaults can be restored by simply deleting this file and all drop-ins.
#
# Use 'systemd-analyze cat-config systemd/resolved.conf' to display the full config.
#
# See resolved.conf(5) for details.

[Resolve]
# Some examples of DNS servers which may be used for DNS= and FallbackDNS=:
# Cloudflare: 1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com 2606:4700:4700::1111#cloudflare-dns.com 2606:4700:4700::1001#cloudflare-dns.com
# Google:     8.8.8.8#dns.google 8.8.4.4#dns.google 2001:4860:4860::8888#dns.google 2001:4860:4860::8844#dns.google
# Quad9:      9.9.9.9#dns.quad9.net 149.112.112.112#dns.quad9.net 2620:fe::fe#dns.quad9.net 2620:fe::9#dns.quad9.net
DNS=8.8.8.8 8.8.4.4
FallbackDNS=1.1.1.1 1.0.0.1
#Domains=
#DNSSEC=no
#DNSOverTLS=no
#MulticastDNS=no
#LLMNR=no
#Cache=no-negative
#CacheFromLocalhost=no
#DNSStubListener=yes
#DNSStubListenerExtra=
#ReadEtcHosts=yes
#ResolveUnicastSingleLabel=no
EOF

```

Querying Metadata inside the VM

```
MMDS_IPV4_ADDR=169.254.169.254
MMDS_TOKEN=$(curl -X PUT "http://${MMDS_IPV4_ADDR}/latest/api/token" \
      -H "X-metadata-token-ttl-seconds: 21600")

META_DATA=$(curl -s -H "Accept: application/json" 169.254.169.254 \
   -H "X-metadata-token: ${MMDS_TOKEN}")

```

Resize rootfs

```
e2fsck -f ubuntu-22.04.ext4
resize2fs ubuntu-22.04.ext4 8G

```

#### Issue

```
Failed handler "fcinit.SetupNetwork": failed to initialize netns: path "/var/lib/netns" does not appear to be a mounted netns: unknown FS magic on "/var/lib/netns": ef53 
2023-11-27T13:23:24.707-0300 [ERROR] run: firecracker VMM did not start, run failed: reason="failed to start machine: failed to initialize netns: path \"/var/lib/netns\" does not appear to be a mounted netns: unknown FS magic on \"/var/lib/netns\": ef53"

Solution remove the /var/lib/netns folder

sudo rm /var/lib/netns

```

```
The stop command will not work in ARM, you can send the kill signal to the PID and stop it that way.
```


## Well Know Paths

### Jailer Path

```
/srv/jailer
```

### CNI Network ip path

```
ls /var/lib/cni/networks/open-fire/
```

# Docker

### Build Image

```
sudo docker build -t open-fire:0.0.1 -f ./open-fire/Dockerfile .
```

### Run Image

```
docker run --name open-fire-1 -p 80:80 open-fire:0.0.1
```

## If you want to compile tools yourself

## CNI Tools
```
https://github.com/containernetworking/plugins
https://github.com/awslabs/tc-redirect-tap
```
You will need to put them in the `/opt/cni` folder

## Kernels
Under the ./host_setup/kernel-config folder there is a config with the network and iptables enabled

### Source of some code and useful information

https://gruchalski.com  
https://jvns.ca/
