#!/usr/bin/env bash

cd bin/

declare -A pack_bins=(
    [gobolt]="gobolt-windows_x86_64.zip"
    [gobolt-linux-amd64]="gobolt-linux-amd64.zip"
    [gobolt-linux-arm64]="gobolt-linux-arm64.zip"
    [AsciiMotionPlayer.exe]="AsciiMotionPlayer_windows_x86_64.zip"
    )

for my_bin in "${!pack_bins[@]}" ; do
    zip "${pack_bins[$my_bin]}" "$my_bin"
done

echo "all done!"

