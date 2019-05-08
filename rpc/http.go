package rpc

import (
	"net"

	log "github.com/inconshreveable/log15"
)

// StartHTTP starts the HTTP RPC endpoint
func StartHTTP(endpoint string, apis []API, modules []string) (net.Listener, *Server, error) {
	// Generate the whitelist based on the allowed modules

	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0) {
			if err := handler.RegisterService(api.Service, api.Namespace); err != nil {
				return nil, nil, err
			}
			log.Debug("HTTP registered", "namespace", api.Namespace)
		}
	}

	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}

	return listener, handler, err
}
