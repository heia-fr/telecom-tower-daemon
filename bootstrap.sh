#!/usr/bin/env bash

apt-get update
apt-get install -y git
apt-get install -y gcc-arm-linux-gnueabihf binutils-arm-linux-gnueabihf
apt-get install -y scons

# Install the latest go
echo "Installing GO... Please wait"
wget -q -O - https://storage.googleapis.com/golang/go1.6.3.linux-amd64.tar.gz \
| tar -C /usr/local -xzf -

echo "Done!"
echo "export PATH=\$PATH:/usr/local/go/bin" >> ~vagrant/.profile
