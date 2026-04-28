#!/usr/bin/env ruby
# usage: ruby rename_images.rb <directory> <optional_suffix>
# example: ruby rename_images.rb ./test_images 2222222222
# usage: ruby scripts/rename_images.rb <directory> <suffix> [master_list_file]
# example: ruby scripts/rename_images.rb ./test_images 2222222222 ./test_data/artists_master.txt

dir = ARGV[0]
suffix = ARGV[1]
master_list_path = ARGV[2]

if dir.nil? || suffix.nil? || !Dir.exist?(dir)
  puts "Usage: ruby scripts/rename_images.rb <directory> <suffix> [master_list_file]"
  exit 1
end

rename_map = {}

Dir.glob(File.join(dir, "*.jpg")).each do |old_path|
  filename = File.basename(old_path)

  # Skip if the filename already contains a hyphen (it already has a cache-busting suffix)
  next if filename.include?("-")

  # Strictly match filenames that are ONLY digits followed by .jpg (e.g., 12.jpg)
  # The ^ and $ anchors ensure the entire filename is matched from start to end.
  if filename =~ /^(\d+)\.jpg$/
    id = $1
    new_filename = "#{id}-#{suffix}.jpg"
    new_path = File.join(dir, new_filename)

    unless old_path == new_path
      File.rename(old_path, new_path)
      puts "Renamed: #{filename} -> #{new_filename}"
      rename_map[filename] = new_filename
    end
  end
end

if master_list_path && File.exist?(master_list_path) && !rename_map.empty?
  content = File.read(master_list_path)
  updated = false

  rename_map.each do |old_name, new_name|
    # Update references in the master list file (e.g., t:1.jpg -> t:1-2222222222.jpg)
    if content.include?("t:#{old_name}")
      content.gsub!("t:#{old_name}", "t:#{new_name}")
      updated = true
    end
  end

  if updated
    File.write(master_list_path, content)
    puts "Updated references in #{master_list_path}"
  end
end
