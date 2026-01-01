# Models module - tests classes, modules, and mixins.
# Tests: class extraction, module extraction, method visibility.

# Log level constants
module LogLevel
  DEBUG = 0
  INFO = 1
  WARN = 2
  ERROR = 3
end

# HTTP method constants
module HttpMethod
  GET = 'GET'
  POST = 'POST'
  PUT = 'PUT'
  DELETE = 'DELETE'
  PATCH = 'PATCH'
end

# Configuration class - tests class with attr_accessor
class Config
  attr_accessor :host, :port, :log_level

  def initialize(host = 'localhost', port = 8080, log_level = 'info')
    @host = host
    @port = port
    @log_level = log_level
  end

  def validate
    return false if host.nil? || host.empty?
    return false if port <= 0 || port > 65_535
    true
  end

  def clone
    Config.new(host, port, log_level)
  end
end

# Handler module - tests module as interface
module Handler
  def handle(input)
    raise NotImplementedError, 'Subclass must implement handle'
  end

  def name
    raise NotImplementedError, 'Subclass must implement name'
  end
end

# Logger class
class Logger
  attr_reader :prefix
  attr_accessor :level

  def initialize(prefix)
    @prefix = prefix
    @level = LogLevel::INFO
  end

  def info(message)
    puts "[INFO] #{@prefix}: #{message}"
  end

  # DEAD CODE
  def debug(message)
    puts "[DEBUG] #{@prefix}: #{message}" if @level <= LogLevel::DEBUG
  end

  # DEAD CODE
  def error(message)
    puts "[ERROR] #{@prefix}: #{message}"
  end
end

# Server class - tests class with methods
class Server
  attr_reader :config

  def initialize(config)
    @config = config
    @running = false
    @logger = Logger.new('server')
  end

  def start
    @running = true
    @logger.info("Starting server on #{@config.host}:#{@config.port}")
    listen
  end

  def stop
    @running = false
    @logger.info('Stopping server')
  end

  def running?
    @running
  end

  private

  def listen
    # Simulated listening
  end

  # DEAD CODE
  def handle_connection(connection)
    # Handle connection
  end
end

# Echo handler - DEAD CODE
class EchoHandler
  include Handler

  def handle(input)
    input
  end

  def name
    'echo'
  end
end

# Upper handler - DEAD CODE
class UpperHandler
  include Handler

  def handle(input)
    input.upcase
  end

  def name
    'upper'
  end
end

# Generic container class
class Container
  def initialize
    @items = []
  end

  def add(item)
    @items << item
  end

  def get(index)
    @items[index]
  end

  def all
    @items.dup
  end

  def size
    @items.size
  end

  def map(&block)
    Container.new.tap do |c|
      @items.each { |item| c.add(block.call(item)) }
    end
  end
end

# Pair class - DEAD CODE
class Pair
  attr_accessor :first, :second

  def initialize(first, second)
    @first = first
    @second = second
  end
end

# Cache class - DEAD CODE
class Cache
  def initialize
    @data = {}
  end

  def set(key, value)
    @data[key] = value
  end

  def get(key)
    @data[key]
  end

  def delete(key)
    @data.delete(key)
  end

  def clear
    @data.clear
  end
end
