//go:build testing
// +build testing

package hub

import "beszel/internal/hub/systems"

// TESTING ONLY: GetSystemManager returns the system manager
func (h *Hub) GetSystemManager() *systems.SystemManager {
	return h.sm
}


// TESTING ONLY: SetAuthKey sets the authentication key
func (h *Hub) SetAuthKey(authKey string) {
	h.authKey = authKey
}
