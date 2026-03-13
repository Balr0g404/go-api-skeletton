package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/response"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and return JWT tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      services.RegisterInput  true  "Registration data"
// @Success      201   {object}  response.APIResponse{data=AuthResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      409   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input services.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, tokens, err := h.authService.Register(input)
	if err != nil {
		switch err {
		case services.ErrEmailAlreadyExists:
			response.Conflict(c, "email already exists")
		default:
			response.InternalError(c)
		}
		return
	}

	response.Created(c, AuthResponse{User: *user, Tokens: *tokens})
}

// Login godoc
// @Summary      Login
// @Description  Authenticate with email and password, returns JWT tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      services.LoginInput  true  "Login credentials"
// @Success      200   {object}  response.APIResponse{data=AuthResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input services.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, tokens, err := h.authService.Login(input)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			response.Unauthorized(c, "invalid email or password")
		case services.ErrAccountDisabled:
			response.Forbidden(c, "account is disabled")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, AuthResponse{User: *user, Tokens: *tokens})
}

// RefreshToken godoc
// @Summary      Refresh tokens
// @Description  Get a new token pair using a valid refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      services.RefreshInput  true  "Refresh token"
// @Success      200   {object}  response.APIResponse{data=TokensResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input services.RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := h.authService.RefreshTokens(input)
	if err != nil {
		response.Unauthorized(c, "invalid refresh token")
		return
	}

	response.OK(c, TokensResponse{Tokens: *tokens})
}

// Logout godoc
// @Summary      Logout
// @Description  Revoke current access token and optionally the refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      LogoutInput  false  "Refresh token to revoke"
// @Success      200   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	accessToken := middleware.GetAccessToken(c)

	var body LogoutInput
	c.ShouldBindJSON(&body)

	h.authService.Logout(accessToken, body.RefreshToken)
	response.Message(c, "logged out successfully")
}

// GetProfile godoc
// @Summary      Get current user profile
// @Description  Returns the authenticated user's profile
// @Tags         Profile
// @Produce      json
// @Success      200  {object}  response.APIResponse{data=models.UserResponse}
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.OK(c, user)
}

// UpdateProfile godoc
// @Summary      Update current user profile
// @Description  Update first name and/or last name
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Param        body  body      services.UpdateProfileInput  true  "Profile fields to update"
// @Success      200   {object}  response.APIResponse{data=models.UserResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input services.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.authService.UpdateProfile(userID, input)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, user)
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the authenticated user's password
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Param        body  body      services.ChangePasswordInput  true  "Password change data"
// @Success      200   {object}  response.APIResponse
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /profile/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input services.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.authService.ChangePassword(userID, input); err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			response.BadRequest(c, "current password is incorrect")
		default:
			response.InternalError(c)
		}
		return
	}

	response.Message(c, "password changed successfully")
}

// ListUsers godoc
// @Summary      List all users
// @Description  Paginated list of users (admin only)
// @Tags         Admin
// @Produce      json
// @Param        page       query     int  false  "Page number"      default(1)
// @Param        page_size  query     int  false  "Items per page"   default(20)
// @Success      200        {object}  response.APIResponse{data=UserListResponse}
// @Failure      401        {object}  response.APIResponse
// @Failure      403        {object}  response.APIResponse
// @Failure      500        {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /admin/users [get]
func (h *AuthHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.authService.ListUsers(page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, UserListResponse{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// SetUserRole godoc
// @Summary      Set user role
// @Description  Change a user's role (admin only)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id    path      int           true  "User ID"
// @Param        body  body      SetRoleInput  true  "New role"
// @Success      200   {object}  response.APIResponse{data=models.UserResponse}
// @Failure      400   {object}  response.APIResponse
// @Failure      401   {object}  response.APIResponse
// @Failure      403   {object}  response.APIResponse
// @Failure      404   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /admin/users/{id}/role [put]
func (h *AuthHandler) SetUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var input SetRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if input.Role != models.RoleUser && input.Role != models.RoleAdmin {
		response.BadRequest(c, "invalid role")
		return
	}

	user, err := h.authService.SetUserRole(uint(id), input.Role)
	if err != nil {
		switch err {
		case services.ErrUserNotFound:
			response.NotFound(c, "user not found")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, user)
}