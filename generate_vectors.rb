#!/usr/bin/env ruby

DESC = <<EOS
Reads artists_master.txt, fetches embedding vectors from Gemini API for the 
features ('f:') field, and writes them to artists_feature_vectors.txt.

Requires GEMINI_API_KEY environment variable.

    -h, --help                       display this help and exit
EOS

program_name = File.basename $PROGRAM_NAME
print_usage = "Usage: #{program_name} [OPTIONS]\n" + DESC

if ARGV.include?("-h") || ARGV.include?("--help")
  puts print_usage
  exit 0
end

require 'net/http'
require 'json'
require 'uri'

API_KEY = ENV['GEMINI_API_KEY']
if API_KEY.nil? || API_KEY.empty?
  STDERR.puts "Error: GEMINI_API_KEY environment variable is not set."
  exit 1
end

MASTER_FILE = "test_data/artists_master.txt"
OUTPUT_FILE = "test_data/artists_feature_vectors.txt"

def get_vector(text)
  uri = URI.parse("https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent?key=#{API_KEY}")
  header = { 'Content-Type' => 'application/json' }
  payload = {
    model: "models/gemini-embedding-001",
    content: {
      parts: [{ text: text }]
    }
  }

  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = true
  request = Net::HTTP::Post.new(uri.request_uri, header)
  request.body = payload.to_json

  response = http.request(request)
  
  if response.code == "200"
    result = JSON.parse(response.body)
    return result.dig("embedding", "values")
  else
    STDERR.puts "API Error: #{response.code} - #{response.body}"
    return nil
  end
end

unless File.exist?(MASTER_FILE)
  STDERR.puts "Error: #{MASTER_FILE} not found."
  exit 1
end

content = File.read(MASTER_FILE)
blocks = content.split("\n\n")

File.open(OUTPUT_FILE, "w") do |f_out|
  blocks.each do |block|
    lines = block.split("\n")
    id = nil
    features = nil

    lines.each do |line|
      if line.start_with?("id:")
        id = line[3..-1].strip
      elsif line.start_with?("f:")
        features = line[2..-1].strip
      end
    end

    if id && features && !features.empty?
      puts "Processing ID: #{id}..."
      vector = get_vector(features)
      if vector
        f_out.write("id:#{id}\n")
        f_out.write("ef:#{vector.join(',')}\n\n")
      end
      # Small sleep to avoid hitting rate limits too hard
      sleep 0.5
    end
  end
end

puts "Done. Vectors written to #{OUTPUT_FILE}"
