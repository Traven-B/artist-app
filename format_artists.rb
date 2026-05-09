#!/usr/bin/env ruby

require 'optimist'

opts = Optimist::options do
  opt :grep, "Output full record lines from artists_augmented.rb", :short => "-g", :default => false
end

names = []

# Process the piped input from more_similar_artists.rb
$stdin.each_line do |line|
  line = line.strip
  # Capture the seed artist from "Target: Name"
  if line =~ /^Target:\s+(.+)$/
    names << $1.strip
  # Capture results from "Name => score"
  elsif line =~ /^(.+?)\s+=>\s+\d/
    names << $1.strip
  end
end

if opts[:grep]
  # Find and print the raw lines from the data file containing the names
  data_lines = File.readlines("artists_augmented.rb")
  names.each do |name|
    # Look for the name in quotes to match exactly within the hash structure
    match = data_lines.find { |l| l.include?("\"#{name}\"") || l.include?("'#{name}'") }
    puts match.strip if match
  end
else
  # Default formatting: art by Name,
  names.each do |name|
    puts "art by #{name},"
  end
end
