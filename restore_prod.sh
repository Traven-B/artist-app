#!/bin/bash
# **** copy from backup_ to _  ****
# remove destinations

echo "⚠️  OVERWRITE PROD? (y/N)"
read -r reply
[[ $reply =~ ^[Yy]$ ]] || { echo "Aborted"; exit 0; }


rm -rf data images
# create the destinations
mkdir -p data images
cp backup_data/* data/
cp backup_images/* images/
echo "prod restored clean:"
ls data/ images/
