#!/usr/bin/env ruby

require "set"

# Load the dataset from augmented file
eval(File.read("artists_augmented.rb"))

# Configuration for matching (copied from similar_artists.rb)
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
  # Pre-processing requirements for :feature
  t = text.to_s.downcase.gsub("|", " ").gsub("unknown", "")
  t.gsub!(/[^a-z0-9\s]/, " ") # remove punctuation

  # Bind phrases together
  PHRASES.each { |p| t.gsub!(p, p.gsub(' ', '_')) }
  
  t.split
   .reject { |w| STOPWORDS.include?(w) || w.length < 3 }
   .map { |w| w.gsub(/(ing|ism|ist|ed|s)$/, '') } # Lightweight stemming
end

# 1. Precompute Inverse Document Frequency (IDF) using :feature
def compute_idf(all_artists)
  total = all_artists.size.to_f
  df = Hash.new(0)
  
  all_artists.each do |a|
    tokenize(a[:feature]).uniq.each { |word| df[word] += 1 }
  end
  
  idf = {}
  df.each { |word, count| idf[word] = Math.log(total / count) }
  idf
end

IDF_TABLE = compute_idf(ARTISTS)

# 2. Convert each artist :feature into a TF-IDF vector
def get_vector(artist)
  tokens = tokenize(artist[:feature])
  counts = Hash.new(0)
  tokens.each { |t| counts[t] += 1 }
  
  vector = {}
  counts.each do |word, freq|
    vector[word] = freq * (IDF_TABLE[word] || 1.0)
  end
  vector
end

# Pre-calculate vectors for all artists once
ARTIST_VECTORS = ARTISTS.map { |a| [a[:id], get_vector(a)] }.to_h

# 3. Calculate Cosine Similarity between two vectors
def similarity(vec_a, vec_b)
  intersection = vec_a.keys & vec_b.keys
  return 0.0 if intersection.empty?

  dot_product = intersection.reduce(0.0) { |sum, w| sum + (vec_a[w] * vec_b[w]) }
  
  mag_a = Math.sqrt(vec_a.values.reduce(0.0) { |sum, v| sum + v**2 })
  mag_b = Math.sqrt(vec_b.values.reduce(0.0) { |sum, v| sum + v**2 })
  
  return 0.0 if mag_a == 0 || mag_b == 0
  dot_product / (mag_a * mag_b)
end

def top_similar(artist, all, k = 5)
  vec_target = ARTIST_VECTORS[artist[:id]]
  
  all
    .reject { |a| a[:id] == artist[:id] }
    .map { |other| [other, similarity(vec_target, ARTIST_VECTORS[other[:id]])] }
    .sort_by { |_, score| -score }
    .first(k)
end

# CLI logic
if ARGV.empty?
  exit 1
end

target_id = ARGV[0]
target = ARTISTS.find { |a| a[:id] == target_id }

if target.nil?
  puts "Artist not found"
  exit 1
end

puts "Target: #{target[:name]}"

top_similar(target, ARTISTS, 5).each do |a, score|
  puts "#{a[:name]} => #{score.round(3)}"
end
