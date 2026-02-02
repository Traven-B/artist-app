#!/bin/bash
# backup prod, data is source, backup_data is dest
rm -rf backup_data backup_images
mkdir -p backup_data backup_images
cp data/* backup_data/
cp images/* backup_images/
echo "prod backed up clean:"
ls backup_data/ backup_images/
