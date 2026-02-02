#!/bin/bash
# **** copy from backup_test_ to test_  ****
# remove destinations

# echo "⚠️  OVERWRITE TEST? (y/N)"
# read -r reply
# [[ $reply =~ ^[Yy]$ ]] || { echo "Aborted"; exit 0; }

rm -rf test_data test_images
# create the destinations
mkdir -p test_data test_images
cp backup_test_data/* test_data/
cp backup_test_images/* test_images/
echo "Test restored clean:"
ls test_data/ test_images/
