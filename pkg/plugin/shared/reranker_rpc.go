package shared

import (
	"net/rpc"
)

// RerankerRPCClient is the RPC client for reranker providers.
type RerankerRPCClient struct {
	client *rpc.Client
}

// Name returns the provider name.
func (c *RerankerRPCClient) Name() string {
	var resp string
	err := c.client.Call("Plugin.Name", new(interface{}), &resp)
	if err != nil {
		return ""
	}
	return resp
}

// RerankArgs are the arguments for the Rerank RPC call.
type RerankArgs struct {
	Query     string
	Documents []string
}

// RerankReply is the reply for the Rerank RPC call.
type RerankReply struct {
	Results []RerankResult
	Error   string
}

// Rerank reranks the documents for the given query.
func (c *RerankerRPCClient) Rerank(query string, documents []string) ([]RerankResult, error) {
	var resp RerankReply
	err := c.client.Call("Plugin.Rerank", &RerankArgs{Query: query, Documents: documents}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &PluginError{Message: resp.Error}
	}
	return resp.Results, nil
}

// MaxDocuments returns the maximum number of documents.
func (c *RerankerRPCClient) MaxDocuments() int {
	var resp int
	err := c.client.Call("Plugin.MaxDocuments", new(interface{}), &resp)
	if err != nil {
		return 100
	}
	return resp
}

// Warmup warms up the provider.
func (c *RerankerRPCClient) Warmup() error {
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
func (c *RerankerRPCClient) Close() error {
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

// RerankerRPCServer is the RPC server for reranker providers.
type RerankerRPCServer struct {
	Impl RerankerProvider
}

// Name returns the provider name.
func (s *RerankerRPCServer) Name(args interface{}, resp *string) error {
	*resp = s.Impl.Name()
	return nil
}

// Rerank reranks the documents for the given query.
func (s *RerankerRPCServer) Rerank(args *RerankArgs, resp *RerankReply) error {
	results, err := s.Impl.Rerank(args.Query, args.Documents)
	if err != nil {
		resp.Error = err.Error()
		return nil
	}
	resp.Results = results
	return nil
}

// MaxDocuments returns the maximum number of documents.
func (s *RerankerRPCServer) MaxDocuments(args interface{}, resp *int) error {
	*resp = s.Impl.MaxDocuments()
	return nil
}

// Warmup warms up the provider.
func (s *RerankerRPCServer) Warmup(args interface{}, resp *string) error {
	err := s.Impl.Warmup()
	if err != nil {
		*resp = err.Error()
	}
	return nil
}

// Close closes the provider.
func (s *RerankerRPCServer) Close(args interface{}, resp *string) error {
	err := s.Impl.Close()
	if err != nil {
		*resp = err.Error()
	}
	return nil
}
