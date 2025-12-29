package shared

import (
	"net/rpc"
)

// EmbeddingRPCClient is the RPC client for embedding providers.
type EmbeddingRPCClient struct {
	client *rpc.Client
}

// Name returns the provider name.
func (c *EmbeddingRPCClient) Name() string {
	var resp string
	err := c.client.Call("Plugin.Name", new(interface{}), &resp)
	if err != nil {
		return ""
	}
	return resp
}

// EmbedArgs are the arguments for the Embed RPC call.
type EmbedArgs struct {
	Texts []string
}

// EmbedReply is the reply for the Embed RPC call.
type EmbedReply struct {
	Embeddings [][]float32
	Error      string
}

// Embed generates embeddings for the given texts.
func (c *EmbeddingRPCClient) Embed(texts []string) ([][]float32, error) {
	var resp EmbedReply
	err := c.client.Call("Plugin.Embed", &EmbedArgs{Texts: texts}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &PluginError{Message: resp.Error}
	}
	return resp.Embeddings, nil
}

// Dimensions returns the embedding dimensions.
func (c *EmbeddingRPCClient) Dimensions() int {
	var resp int
	err := c.client.Call("Plugin.Dimensions", new(interface{}), &resp)
	if err != nil {
		return 0
	}
	return resp
}

// MaxBatchSize returns the maximum batch size.
func (c *EmbeddingRPCClient) MaxBatchSize() int {
	var resp int
	err := c.client.Call("Plugin.MaxBatchSize", new(interface{}), &resp)
	if err != nil {
		return 1
	}
	return resp
}

// Warmup warms up the provider.
func (c *EmbeddingRPCClient) Warmup() error {
	var resp string
	err := c.client.Call("Plugin.Warmup", new(interface{}), &resp)
	if err != nil {
		return err
	}
	if resp != "" {
		return &PluginError{Message: resp}
	}
	return nil
}

// Close closes the provider.
func (c *EmbeddingRPCClient) Close() error {
	var resp string
	err := c.client.Call("Plugin.Close", new(interface{}), &resp)
	if err != nil {
		return err
	}
	if resp != "" {
		return &PluginError{Message: resp}
	}
	return nil
}

// EmbeddingRPCServer is the RPC server for embedding providers.
type EmbeddingRPCServer struct {
	Impl EmbeddingProvider
}

// Name returns the provider name.
func (s *EmbeddingRPCServer) Name(args interface{}, resp *string) error {
	*resp = s.Impl.Name()
	return nil
}

// Embed generates embeddings for the given texts.
func (s *EmbeddingRPCServer) Embed(args *EmbedArgs, resp *EmbedReply) error {
	embeddings, err := s.Impl.Embed(args.Texts)
	if err != nil {
		resp.Error = err.Error()
		return nil
	}
	resp.Embeddings = embeddings
	return nil
}

// Dimensions returns the embedding dimensions.
func (s *EmbeddingRPCServer) Dimensions(args interface{}, resp *int) error {
	*resp = s.Impl.Dimensions()
	return nil
}

// MaxBatchSize returns the maximum batch size.
func (s *EmbeddingRPCServer) MaxBatchSize(args interface{}, resp *int) error {
	*resp = s.Impl.MaxBatchSize()
	return nil
}

// Warmup warms up the provider.
func (s *EmbeddingRPCServer) Warmup(args interface{}, resp *string) error {
	err := s.Impl.Warmup()
	if err != nil {
		*resp = err.Error()
	}
	return nil
}

// Close closes the provider.
func (s *EmbeddingRPCServer) Close(args interface{}, resp *string) error {
	err := s.Impl.Close()
	if err != nil {
		*resp = err.Error()
	}
	return nil
}

// PluginError represents an error from a plugin.
type PluginError struct {
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}
