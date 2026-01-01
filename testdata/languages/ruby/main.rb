# Main module demonstrating various Ruby patterns for parser testing.
# Tests: entry points, classes, modules, methods.

require_relative 'utils'
require_relative 'models'

# Constants - tests constant extraction
MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30
APP_NAME = 'TestApp'

# Module-level variables
$app_version = '1.0.0'
$initialized = false

# Main entry point - should be marked as reachable.
def main
  puts "Starting #{APP_NAME}"

  # Function calls - tests reference extraction
  config = load_config
  unless initialize_app(config)
    warn 'Initialization failed'
    exit 1
  end

  # Create and start server
  server = Server.new(config)
  server.start

  # Using utility functions
  items = %w[a b c]
  result = Utils.process_data(items)
  output = Utils.format_output(result)
  puts output

  # Calling transitive functions
  run_pipeline

  # Cleanup
  server.stop
end

# Load configuration - called from main, should be reachable.
def load_config
  Config.new('localhost', 8080, 'info')
end

# Initialize application - called from main, should be reachable.
def initialize_app(config)
  setup_logging(config.log_level)
  $initialized = true
  true
end

# Internal helper - called from initialize, should be reachable.
def setup_logging(level)
  puts "Setting log level to: #{level}"
end

# Orchestrate data pipeline - tests transitive reachability.
def run_pipeline
  data = fetch_data
  transformed = transform_data(data)
  save_data(transformed)
end

# Fetch data - called by run_pipeline, should be reachable.
def fetch_data
  'sample data'
end

# Transform data - called by run_pipeline, should be reachable.
def transform_data(data)
  "transformed: #{data}"
end

# Save data - called by run_pipeline, should be reachable.
def save_data(data)
  puts "Saving: #{data}"
end

# ============================================================================
# Dead code section - methods that are never called
# ============================================================================

# This method is never called - DEAD CODE.
def unused_method
  puts 'This is never executed'
end

# Also never called - DEAD CODE.
def another_unused
  'dead'
end

# Starts a chain of dead code - DEAD CODE.
def dead_chain_start
  dead_chain_middle
end

# In the middle of dead chain - DEAD CODE (transitive).
def dead_chain_middle
  dead_chain_end
end

# End of dead chain - DEAD CODE (transitive).
def dead_chain_end
  puts 'End of dead chain'
end

# Run main
main if __FILE__ == $PROGRAM_NAME
