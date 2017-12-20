#!/bin/bash

export BOOK_DIR=/home/erwin/Downloads/drive-download-20171002T115616Z-001

./build.sh

echo "running now"
export ALLOW_DELETES=true
./booksing
