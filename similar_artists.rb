#!/usr/bin/env ruby

require "set"

# Load the dataset
ARTISTS = eval(File.read("artists_data.rb"))

# Configuration for better semantic matching
PHRASES = [
  "art nouveau", "art deco", "woodblock print", "mid century", "oil on canvas",
  "city pop", "post impressionist", "shin hanga", "ukiyo e", "digital painting",
  "fine art", "fashion illustration", "concept artist", "graphic designer",
  "comic book", "street art", "pop surrealism", "new objectivity", "lowbrow art"
]

STOPWORDS = Set.new([
  "the", "and", "of", "in", "a", "to", "with", "is", "for", "as", "by", "on",
  "artist", "painter", "illustrator", "born", "known", "style", "work", "his",
  "her", "was", "an", "who", "which", "from", "at", "also", "its", "their",
  "one", "featured", "including", "depicting", "characterized", "often", "use",
  "used", "using", "into", "features", "through", "well", "became", "both"
])

def tokenize(text)
  t = text.downcase
  
  # Bind phrases together so they are treated as single tokens
  PHRASES.each { |p| t.gsub!(p, p.gsub(' ', '_')) }
  
  t.gsub(/[^a-z0-9_\s]/, " ")
   .split
   .reject { |w| STOPWORDS.include?(w) || w.length < 3 }
   .map { |w| w.gsub(/(ing|ism|ist|ed|s)$/, '') } # Lightweight stemming
end

# 1. Precompute Inverse Document Frequency (IDF)
# This devalues common words like "painting" and boosts specific words like "neofuturism"
def compute_idf(all_artists)
  total = all_artists.size.to_f
  df = Hash.new(0)
  
  all_artists.each do |a|
    tokenize(a[:desc]).uniq.each { |word| df[word] += 1 }
  end
  
  idf = {}
  df.each { |word, count| idf[word] = Math.log(total / count) }
  idf
end

IDF_TABLE = compute_idf(ARTISTS)

# 2. Convert each artist description into a TF-IDF vector
def get_vector(artist)
  tokens = tokenize(artist[:desc])
  counts = Hash.new(0)
  tokens.each { |t| counts[t] += 1 }
  
  vector = {}
  counts.each do |word, freq|
    # TF * IDF
    vector[word] = freq * (IDF_TABLE[word] || 1.0)
  end
  vector
end

# Pre-calculate vectors for all artists once
ARTIST_VECTORS = ARTISTS.map { |a| [a[:id], get_vector(a)] }.to_h

# 3. Calculate Cosine Similarity between two vectors
# Measures the angle between descriptions; length-independent
def similarity(vec_a, vec_b)
  intersection = vec_a.keys & vec_b.keys
  return 0.0 if intersection.empty?

  dot_product = intersection.reduce(0.0) { |sum, w| sum + (vec_a[w] * vec_b[w]) }
  
  mag_a = Math.sqrt(vec_a.values.reduce(0.0) { |sum, v| sum + v**2 })
  mag_b = Math.sqrt(vec_b.values.reduce(0.0) { |sum, v| sum + v**2 })
  
  return 0.0 if mag_a == 0 || mag_b == 0
  dot_product / (mag_a * mag_b)
end

def top_similar(artist, all, k = 3)
  vec_target = ARTIST_VECTORS[artist[:id]]
  
  all
    .reject { |a| a[:id] == artist[:id] }
    .map { |other| [other, similarity(vec_target, ARTIST_VECTORS[other[:id]])] }
    .sort_by { |_, score| -score }
    .first(k)
end

# Example usage (matching simple.rb target):
target = ARTISTS[118] # Gwen Fremlin
puts "Target: #{target[:name]}"
puts "Description: #{target[:desc]}"
puts "-" * 40

top_similar(target, ARTISTS).each do |a, score|
  puts "#{a[:name]} (Score: #{score.round(3)})"
  puts "  #{a[:desc][0..100]}..."
end
