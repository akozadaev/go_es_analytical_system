package auth

// HasMapIndexerAccess — доступ к инструменту js_API_Ya_map (админы).
// Учитываются роли из go_oauth2_server (ROLE_*) и альтернативные имена.
func HasMapIndexerAccess(roles []string) bool {
	for _, r := range roles {
		switch r {
		case "ROLE_ADMIN", "ROLE_SUPER_ADMIN", "admin_role", "super_admin_role":
			return true
		}
	}
	return false
}
