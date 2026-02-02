#!/usr/bin/env ruby
# old_to_new.rb

# One-off migration script: old artist format → current master format.
# Kept as a reference/template for future format changes.

require "fileutils"

SRC_MASTER = "../art/artists.txt"       # Old format in old project root
DST_MASTER = "data/artists_master.txt"  # New format
OLD_IMAGES = "old_images"         # Source thumbs
NEW_IMAGES = "images"             # Destination (empty)

# Ensure dirs
FileUtils.mkdir_p("data")
FileUtils.mkdir_p(NEW_IMAGES)

artists = []
File.read(SRC_MASTER).scan(/(n:.*?)(?=\n\n|\z)/m) do |block|
  lines = Hash[block[0].scan(/^([ndih]):(.*)$/)]
  next unless lines["n"]

  artists << {
    name: lines["n"].strip,
    desc: lines["d"]&.strip || "",
    img_url: lines["i"]&.strip,
    old_thumb: "#{lines["h"].strip}.jpg",
  }
end

puts "Converting #{artists.size} artists..."

new_content = ""
artists.each_with_index do |artist, i|
  id = i + 1
  new_thumb = "#{id}.jpg"

  # Copy + rename thumb
  old_path = File.join(OLD_IMAGES, artist[:old_thumb])
  new_path = File.join(NEW_IMAGES, new_thumb)

  if File.exist?(old_path)
    FileUtils.cp(old_path, new_path)
    puts "  #{id}: #{artist[:old_thumb]} → #{new_thumb}"
  else
    puts "  ⚠️ #{id}: Missing #{artist[:old_thumb]}"
  end

  # New format
  new_content << "id:#{id}\n"
  new_content << "n:#{artist[:name]}\n"
  new_content << "d:#{artist[:desc]}\n"
  new_content << "i:#{artist[:img_url]}\n"
  new_content << "t:#{new_thumb}\n\n"
end

File.write(DST_MASTER, new_content.strip)
puts "\n✅ Done! #{artists.size} artists → #{DST_MASTER}"
puts "✅ Thumbs → #{NEW_IMAGES}/"
puts "go run main.go → localhost:8080/gallery"
