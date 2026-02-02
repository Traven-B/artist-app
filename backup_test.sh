#!/bin/bash
rm -rf backup_test_data backup_test_images
mkdir -p backup_test_data backup_test_images
cp test_data/* backup_test_data/
cp test_images/* backup_test_images/
echo "Test backed up clean:"
ls backup_test_data/ backup_test_images/
