package jwtutil

import "short-video-platform/pkg/auth"

func Sign(userID, username string) (string, error) {
	return auth.SignWithRole(userID, username, auth.RoleUser)
}

func SignWithRole(userID, username, role string) (string, error) {
	return auth.SignWithRole(userID, username, role)
}

func Parse(tokenStr string) (*auth.Claims, error) {
	return auth.ParseToken(tokenStr)
}
