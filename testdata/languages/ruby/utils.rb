# Utils module - tests module functions and helpers.
# Tests: module methods, blocks, lambdas.

# Utils module with utility functions
module Utils
  module_function

  # Process string data - called from main, should be reachable.
  def process_data(items)
    return '' if items.nil? || items.empty?
    items.map(&:upcase).join(', ')
  end

  # Format output string - called from main, should be reachable.
  def format_output(data)
    "Result: #{data}"
  end

  # Validate configuration - called from various places, should be reachable.
  def validate_config(config)
    config.validate
  end

  # ============================================================================
  # Dead code section - functions that are never called
  # ============================================================================

  # Hash string - DEAD CODE.
  def hash_string(s)
    hash = 5381
    s.each_byte { |c| hash = ((hash << 5) + hash) + c }
    format('%016x', hash & 0xFFFFFFFFFFFFFFFF)
  end

  # Filter strings - DEAD CODE.
  def filter_strings(items, &predicate)
    items.select(&predicate)
  end

  # Map strings - DEAD CODE.
  def map_strings(items, &transform)
    items.map(&transform)
  end

  # Reduce strings - DEAD CODE.
  def reduce_strings(items, seed, &accumulator)
    items.reduce(seed, &accumulator)
  end

  # Chunk array - DEAD CODE.
  def chunk_array(items, size)
    items.each_slice(size).to_a
  end

  # Flatten nested arrays - DEAD CODE.
  def flatten_array(items)
    items.flatten
  end

  # Group by key - DEAD CODE.
  def group_by_key(items, &key_selector)
    items.group_by(&key_selector)
  end

  # Sort by comparator - DEAD CODE.
  def sort_by_comparator(items, &comparator)
    items.sort(&comparator)
  end

  # Distinct by key - DEAD CODE.
  def distinct_by(items, &key_selector)
    items.uniq(&key_selector)
  end

  # Take while predicate - DEAD CODE.
  def take_while_predicate(items, &predicate)
    items.take_while(&predicate)
  end
end

# String extensions module - DEAD CODE
module StringExtensions
  refine String do
    # Check if string is blank - DEAD CODE.
    def blank?
      nil? || strip.empty?
    end

    # Truncate string - DEAD CODE.
    def truncate(max_length)
      return self if length <= max_length
      "#{self[0, max_length]}..."
    end

    # Capitalize each word - DEAD CODE.
    def titleize
      split.map(&:capitalize).join(' ')
    end
  end
end

# Enumerable extensions module - DEAD CODE
module EnumerableExtensions
  # Find first matching or nil - DEAD CODE.
  def find_or_nil(&block)
    find(&block)
  end

  # Safe first element - DEAD CODE.
  def safe_first
    first
  rescue StandardError
    nil
  end
end
