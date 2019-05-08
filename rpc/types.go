package rpc

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Service   interface{} // receiver instance which holds the methods
}
