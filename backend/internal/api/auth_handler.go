package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tomatogether/backend/internal/service"
)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if req.Username == "" || req.Password == "" {
		s.errorResponse(w, http.StatusBadRequest, "username_and_password_required")
		return
	}

	userService := s.userService.(*service.UserService)
	authService := s.authService.(*service.AuthService)

	user, err := userService.Register(req.Username, req.Password)
	if err != nil {
		if err == service.ErrUsernameTaken {
			s.errorResponse(w, http.StatusConflict, "username_taken")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "registration_failed")
		return
	}

	token, err := authService.GenerateAccessToken(user.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "token_generation_failed")
		return
	}

	s.jsonResponse(w, http.StatusCreated, formatSuccess(map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	}))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if req.Username == "" || req.Password == "" {
		s.errorResponse(w, http.StatusBadRequest, "username_and_password_required")
		return
	}

	userService := s.userService.(*service.UserService)
	authService := s.authService.(*service.AuthService)

	user, err := userService.Login(req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			s.errorResponse(w, http.StatusUnauthorized, "invalid_credentials")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "login_failed")
		return
	}

	token, err := authService.GenerateAccessToken(user.ID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "token_generation_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	}))
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	userService := s.userService.(*service.UserService)
	user, err := userService.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		s.errorResponse(w, http.StatusNotFound, "user_not_found")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"has_password": user.IsPersistent(),
		"created_at":  user.CreatedAt.Format(time.RFC3339),
	}))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	authService := s.authService.(*service.AuthService)
	claims, err := authService.ValidateAccessToken(req.RefreshToken)
	if err != nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	accessToken, err := authService.GenerateAccessToken(claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "token_generation_failed")
		return
	}

	expiry := getEnvInt("JWT_ACCESS_TOKEN_EXPIRY", 3600)
	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"access_token": accessToken,
		"expires_in":   expiry,
	}))
}

func (s *Server) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	token := s.getRoomTokenFromRequest(r)
	if token == "" {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	authService := s.authService.(*service.AuthService)
	roomToken, err := authService.ValidateRoomToken(token)
	if err != nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Password == "" {
		s.errorResponse(w, http.StatusBadRequest, "password_required")
		return
	}

	userService := s.userService.(*service.UserService)
	user, err := userService.UpgradeUser(roomToken.UserID, req.Password)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "upgrade_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"message": "已升级为持久化用户",
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
	}))
}

func (s *Server) validateL3Token(r *http.Request) *service.Claims {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		return nil
	}

	authService := s.authService.(*service.AuthService)
	claims, err := authService.ValidateAccessToken(token)
	if err != nil {
		return nil
	}
	return claims
}

func (s *Server) validateL2Token(r *http.Request) string {
	token := s.getRoomTokenFromRequest(r)
	if token == "" {
		return ""
	}

	authService := s.authService.(*service.AuthService)
	roomToken, err := authService.ValidateRoomToken(token)
	if err != nil {
		return ""
	}
	return roomToken.UserID
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}