#!/usr/bin/env ruby

# copy("<!DOCTYPE html>\n" + document.documentElement.outerHTML)

# "Reika Iwami created abstract sōsaku hanga woodblock prints. Her style features monochromatic sumi ink, rich wood grain textures, deep embossing, and gold or silver leaf accents."

require "fileutils"

SOURCE = File.expand_path("~/work/go/toss")
BUILD = File.expand_path("~/work/go/gh_artist_pages")

INDEX = File.join(BUILD, "index.html")
GALLERY = File.join(BUILD, "artist-gallery.html")

puts "== Export setup =="

# 1. reset build dir
FileUtils.rm_rf(BUILD)
FileUtils.mkdir_p(BUILD)

# 2. create empty files (so vim/browser has targets)
File.write(INDEX, "")
File.write(GALLERY, "")

FileUtils.mkdir_p(File.join(BUILD, "static"))
FileUtils.mkdir_p(File.join(BUILD, "images"))

FileUtils.cp_r(File.join(SOURCE, "static"), BUILD)
FileUtils.cp_r(File.join(SOURCE, "test_images"), File.join(BUILD, "images"))

puts "\n== Next steps =="

puts "1. Open build dir:"
puts "   cd #{BUILD}"
puts ""

puts "2. Open files in editor:"
puts "   vim index.html"
puts "   vim artist-gallery.html"
puts ""

puts "3. In browser:"
puts "   - open app index page"
puts "   - complete form"
puts "   - press F12 → copy DOM (outerHTML)"
puts "   - paste into index.html"
puts ""

puts "4. Repeat for gallery page"
puts ""

puts "== Done staging files =="
