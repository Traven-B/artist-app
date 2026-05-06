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

puts <<~MSG

       == Next steps ==
       1. Open build dir:
          cd #{BUILD}

       2. Open files in editor:
          vim index.html
          vim artist-gallery.html

       3. In project dir start webapp in test if you havn's already.
          - ./run.sh test

       4. In browser:
          - open app index page
          - add Reika Iwami from list
          - complete desc with ' Reika Iwami created abstract sōsaku hanga woodblock prints. Her style features monochromatic sumi ink, rich wood grain textures, deep embossing, and gold or silver leaf accents. '
          - press F12 → copy("<!DOCTYPE html>\n" + document.documentElement.outerHTML)
          - paste into index.html

       5. Repeat for gallery page
          - do not select any artits and so do not make a working group at bottom

       == Done staging files ==
     MSG
