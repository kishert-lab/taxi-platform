// Package dispatch contains the asynchronous order dispatch application service.
//
// Dispatch state is stored in Redis and PostgreSQL so the service can run across
// multiple backend instances without relying on process memory.
package dispatch
