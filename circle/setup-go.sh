#!/bin/sh -ex

cache_dir=${CIRCLE_CACHE_DIR:-.}

go_pkg_loc='https://storage.googleapis.com/golang'
go_pkg='go1.7.3.linux-amd64.tar.gz'

sudo rm -rf /usr/local/go

if [ ! -e "$cache_dir/$go_pkg" ] ; then
    curl -o "$cache_dir/$go_pkg" "$go_pkg_loc/$go_pkg"
fi

sudo tar -C /usr/local -xzf "$cache_dir/$go_pkg"
