#!/bin/bash -ex

cache_dir=${CIRCLE_CACHE_DIR:-.}

go_pkg_loc='https://storage.googleapis.com/golang'
go_pkg='go1.7.3.linux-amd64.tar.gz'

go_pkg_cache="$cache_dir/$go_pkg"

sudo rm -rf /usr/local/go

if [ ! -d "$cache_dir" ] ; then
    mkdir "$cache_dir"
fi

if [ ! -e "$go_pkg_cache" ] ; then
    curl -o "$go_pkg_cache" "$go_pkg_loc/$go_pkg"
fi

sudo tar -C /usr/local -xzf "$go_pkg_cache"
