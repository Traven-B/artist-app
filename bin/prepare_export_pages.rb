#!/usr/bin/env ruby

require "fileutils"

SOURCE = File.expand_path("..", __dir__)
BUILD = File.expand_path("../../gh_artist_pages", __dir__) # Sister directory


INDEX = File.join(BUILD, "index.html")
GALLERY = File.join(BUILD, "artist-gallery.html")

def fixup_html(path)
  return unless File.exist?(path) && !File.read(path).empty?
  puts "   Fixing navigation links in #{File.basename(path)}..."
  content = File.read(path)
  content.gsub!('href="/"', 'href="index.html"')
  content.gsub!('href="/gallery"', 'href="artist-gallery.html"')
  File.write(path, content)
end

puts "== Export setup =="

# 1. reset build dir
FileUtils.rm_rf(BUILD)
FileUtils.mkdir_p(BUILD)

# 2. create empty files
File.write(INDEX, "")
File.write(GALLERY, "")

FileUtils.mkdir_p(File.join(BUILD, "static"))
FileUtils.mkdir_p(File.join(BUILD, "images"))

FileUtils.cp_r(File.join(SOURCE, "static"), BUILD)
FileUtils.cp_r(File.join(SOURCE, "test_images"), File.join(BUILD, "images"))

puts <<~MSG

       == Manual Steps ==
       1. Open the running app in your browser (TEST_MODE=true).
       2. For both the Index and Gallery pages, run this in the F12 console:
          copy("<!DOCTYPE html>\\n" + document.documentElement.outerHTML)
       3. Paste the result into:
          Index:   #{INDEX}
          Gallery: #{GALLERY}

       >>> Press [ENTER] once you have saved both files to perform fixups...
     MSG

$stdin.gets

puts "== Running fixups =="
fixup_html(INDEX)
fixup_html(GALLERY)

puts "\nDone. You can now run a_deploy.sh (ensure the path in that script matches #{BUILD})."
