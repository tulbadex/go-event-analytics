package controllers

import (
	"event-analytics/config"
	"event-analytics/models"
	"event-analytics/render"
	"event-analytics/utils"
	"event-analytics/services"
	"net/http"

	"fmt"
	"log"
    "strconv"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	var input struct {
		Identifier string `form:"identifier" binding:"required"` // Changed to a single identifier for username/email
		Password   string `form:"password" binding:"required"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Invalid input",
			"title": "Login",
		})
		return
	}

	var user models.User
	// Check if the identifier is an email or username
	if err := config.DB.Where("username = ? OR email = ?", input.Identifier, input.Identifier).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Invalid credentials",
			"title": "Login",
		})
		return
	}

	if !user.IsVerified {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Please verify your email before logging in.",
			"title": "Login",
		})
		return
	}

	if err := user.CheckPassword(input.Password); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Invalid credentials",
			"title": "Login",
		})
		return
	}

	sessionToken := utils.GenerateRandomToken()
	if err := utils.SaveSession(sessionToken, user); err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error": "Failed to create session",
			"title": "Login",
		})
		return
	}

	c.SetCookie("session_token", sessionToken, 3600, "/", "", false, true)
	c.Redirect(http.StatusFound, "/user/dashboard")
}

func Logout(c *gin.Context) {
	sessionToken, err := c.Cookie("session_token")
	if err != nil {
		log.Printf("Logout: Failed to get session token: %v", err)
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	err = config.RedisClient.Del(c, sessionToken).Err()
	if err != nil {
		log.Printf("Logout: Redis deletion failed: %v", err)
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)
	log.Println("Logout: User successfully logged out")
	c.Redirect(http.StatusFound, "/auth/login")
}

func Register(c *gin.Context) {
	var input struct {
		Username string `form:"username" binding:"required"`
		Email    string `form:"email" binding:"required,email"`
		Password string `form:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Invalid input",
			"title": "Register",
		})
		return
	}

	var user models.User
	if err := config.DB.Where("username = ?", input.Username).First(&user).Error; err == nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Username already taken",
			"title": "Register",
		})
		return
	}

	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err == nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Email already taken",
			"title": "Register",
		})
		return
	}

	user = models.User{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
	}

	if err := user.HashPassword(); err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error": "Failed to hash password",
			"title": "Register",
		})
		return
	}

	tx := config.DB.Begin() // Start a transaction

	if err := tx.Create(&user).Error; err != nil {
        tx.Rollback() // Rollback if user creation fails
        c.HTML(http.StatusInternalServerError, "register.html", gin.H{
            "error": "Failed to create user",
            "title": "Register",
        })
        return
    }

    // Create password history entry
	passwordHistory := models.PasswordHistory{
		UserID:    user.ID,
		Password:  user.Password, // Already hashed
	}

	if err := tx.Create(&passwordHistory).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error": "Failed to create password history",
			"title": "Register",
		})
		return
	}

    // Assign 'user' role to the registered user
    var userRole models.Role
    if err := tx.Where("name = ?", "user").First(&userRole).Error; err != nil {
        tx.Rollback() // Rollback if role assignment fails
        c.HTML(http.StatusInternalServerError, "register.html", gin.H{
            "error": "Failed to assign role.",
            "title": "Register",
        })
        return
    }

    userRoleMapping := models.UserRole{
        UserID: user.ID,
        RoleID: userRole.ID,
    }

    if err := tx.Create(&userRoleMapping).Error; err != nil {
        tx.Rollback() // Rollback if role mapping fails
        c.HTML(http.StatusInternalServerError, "register.html", gin.H{
            "error": "Failed to assign role.",
            "title": "Register",
        })
        return
    }

    // Commit the transaction after all operations are successful
    if err := tx.Commit().Error; err != nil {
        c.HTML(http.StatusInternalServerError, "register.html", gin.H{
            "error": "Unable to complete registration. Please try again.",
            "title": "Register",
        })
        return
    }

	token := utils.GenerateRandomToken()
	if err := config.DB.Exec(
		"INSERT INTO verification_tokens (user_id, token) VALUES (?, ?)", user.ID, token,
	).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error": "Failed to generate verification token",
			"title": "Register"})
		return
	}

	baseURL := utils.GetBaseURL(c.Request)
	verifyURL := fmt.Sprintf("%s/auth/verify?token=%s", baseURL, token)

	body := utils.RenderTemplate("templates/verification.html", map[string]interface{}{
		"VerifyURL": verifyURL,
	})

	utils.SendEmailAsync(c.Request, input.Email, "Verify Your Email", body)

	// c.Redirect(http.StatusFound, "/auth/login")
	c.HTML(http.StatusOK, "register.html", gin.H{
		"success": "User created successfully. A verification email has been sent to your email address. Please verify to complete registration.",
		"title":   "Register",
	})
}

func Verify(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "verify.html", gin.H{
			"error": "Invalid verification token",
			"title": "User account verification",
		})
		return
	}

	var verification struct {
		UserID string
		Token  string
	}
	if err := config.DB.Raw(
		"SELECT user_id, token FROM verification_tokens WHERE token = ?", token,
	).Scan(&verification).Error; err != nil {
		c.HTML(http.StatusBadRequest, "verify.html", gin.H{
			"error": "Invalid or expired verification token",
			"title": "User account verification",
		})
		return
	}

	if err := config.DB.Exec(
		"DELETE FROM verification_tokens WHERE token = ?", token,
	).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "verify.html", gin.H{
			"error": "Failed to validate verification",
			"title": "User account verification",
		})
		return
	}

	// Update the user's verification status
	if err := config.DB.Model(&models.User{}).Where("id = ?", verification.UserID).Update("is_verified", true).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "verify.html", gin.H{
			"error": "Failed to update verification status",
			"title": "User account verification",
		})
		return
	}

	c.HTML(http.StatusOK, "verify.html", gin.H{
		"success": "Your email has been successfully verified. You can now log in.",
		"title":   "User account verification",
	})
}

func ForgotPassword(c *gin.Context) {
	email := c.PostForm("email")
	var user models.User
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusBadRequest, "forgot_password.html", gin.H{
			"error": "Email not found",
			"title": "Forget Password",
		})
		return
	}

	token := utils.GenerateRandomToken()
	config.DB.Exec("INSERT INTO password_resets (email, token) VALUES (?, ?)", email, token)

	baseURL := utils.GetBaseURL(c.Request)
	verifyURL := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, token)

	body := utils.RenderTemplate("templates/password_reset_mail.html", map[string]interface{}{
		"VerifyURL": verifyURL,
	})

	utils.SendEmailAsync(c.Request, email, "Reset your password", body)
	c.HTML(http.StatusOK, "forgot_password.html", gin.H{
		"success": "Reset link sent to your email",
		"title":   "Forget Password",
	})
}

func ResetPassword(c *gin.Context) {
	token := c.PostForm("token")
	newPassword := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	if newPassword != confirmPassword {
        render.Render(c, gin.H{
            "error": "Passwords do not match",
            "title": "Reset Password",
            "token": token,
        }, "reset_password.html")
		return
	}

	// Start transaction
	tx := config.DB.Begin()

	// Get email from reset token
	var reset struct {
		Email string
	}
	if err := tx.Raw("SELECT email FROM password_resets WHERE token = ?", token).Scan(&reset).Error; err != nil {
		tx.Rollback()
		c.Redirect(http.StatusFound, "/auth/login?error=invalid_reset_token")
		c.Abort()
		return
	}

	// Find user by email
	var user models.User
	if err := tx.Where("email = ?", reset.Email).First(&user).Error; err != nil {
		tx.Rollback()
		log.Printf("Reset password: User not found for email %s: %v", reset.Email, err)
		c.Redirect(http.StatusFound, "/auth/login?error=user_not_found")
		c.Abort()
		return
	}

	// Fetch the last 5 password histories
	var passwordHistories []models.PasswordHistory
	if err := tx.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		// Limit(5).
		Find(&passwordHistories).Error; err != nil {
		tx.Rollback()
        render.Render(c, gin.H{
            "error": "Unable to verify password history. Please try again",
            "title": "Reset Password",
            "token": token,
        }, "reset_password.html")
		return
	}

	// Compare the plain new password with previous passwords using the model function
    if user.IsPasswordUsed(newPassword, passwordHistories) {
        tx.Rollback()
        render.Render(c, gin.H{
            "error": "This password has been used recently. Please choose a different password",
            "title": "Reset Password",
            "token": token,
        }, "reset_password.html")
        return
    }

	// Update user's password
	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		tx.Rollback()
		log.Printf("Reset password: Failed to hash password: %v", err)
        render.Render(c, gin.H{
			"error": "An error occurred while updating password",
			"title": "Reset Password",
			"token": token,
		}, "reset_password.html")
		return
	}

	// Save updated user
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		log.Printf("Reset password: Failed to save user: %v", err)
        render.Render(c, gin.H{
			"error": "An error occurred while saving password",
			"title": "Reset Password",
			"token": token,
		}, "reset_password.html")
		return
	}

	// Save the new password to history
	newHistory := models.PasswordHistory{
		UserID:    user.ID,
		Password:  user.Password,
	}

	if err := tx.Create(&newHistory).Error; err != nil {
		tx.Rollback()
		log.Printf("Reset password: Failed to save password history: %v", err)
        render.Render(c, gin.H{
			"error": "An error occurred while saving password history",
			"title": "Reset Password",
			"token": token,
		}, "reset_password.html")
		return
	}

	// Delete the used token
	if err := tx.Exec("DELETE FROM password_resets WHERE token = ?", token).Error; err != nil {
		tx.Rollback()
		log.Printf("Reset password: Failed to delete token: %v", err)
        render.Render(c, gin.H{
			"error": "An error occurred while completing password reset",
			"title": "Reset Password",
			"token": token,
		}, "reset_password.html")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("Reset password: Failed to commit transaction: %v", err)
        render.Render(c, gin.H{
			"error": "An error occurred while finalizing password reset",
			"title": "Reset Password",
			"token": token,
		}, "reset_password.html")
		return
	}

	// Redirect to login with success message
	c.Redirect(http.StatusFound, "/auth/login?success=password_reset")
}

func Dashboard(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists || user == nil {
		c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
		c.Abort()
		return
	}

    currentUser, ok := user.(*models.User)
    if !ok {
        c.Redirect(http.StatusFound, "/auth/login?error=invalid_user")
        c.Abort()
        return
    }

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 12 // Number of events per page
	offset := (page - 1) * limit

	var events []models.Event
	if err := config.DB.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error": "Failed to fetch events",
			"title": "Dashboard",
			"user":  user,
		})
		return
	}

    // Check editability for each event
    for i := range events {
        events[i].IsEditable = services.CheckEventEditPermission(&events[i], currentUser)
    }

	data := gin.H{
		"events": events,
		"title":  "Dashboard",
		"user":   user,
	}

	c.HTML(http.StatusOK, "dashboard.html", data)
}

func EditProfile(c *gin.Context) {
	// Retrieve the user from the session
	user, err := utils.GetUserFromSession(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
		return
	}

	// Parse and validate the input form
	var input struct {
		FirstName string `form:"firstName" binding:"required"`
		LastName  string `form:"lastName" binding:"required"`
		Address   string `form:"address" binding:"required"`
		Username  string `form:"username" binding:"required"`
		Email     string `form:"email" binding:"required,email"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "profile.html", gin.H{
			"error": "Invalid input. Please fill all fields correctly.",
			"user":  user,
			"title": "Profile",
		})
		return
	}

	// Prepare updates
	updates := map[string]interface{}{
		"first_name": input.FirstName,
		"last_name":  input.LastName,
		"address":    input.Address,
		"username":   input.Username,
		"email":      input.Email,
	}

	// Update the user profile
	if err := utils.UpdateUserProfile(user.ID.String(), updates); err != nil {
		c.HTML(http.StatusConflict, "profile.html", gin.H{
			"error": err.Error(),
			"user":  user,
			"title": "Profile",
		})
		return
	}

	// Update the session with the new user data
	user.FirstName = input.FirstName
	user.LastName = input.LastName
	user.Address = input.Address
	user.Username = input.Username
	user.Email = input.Email
	if err := utils.SetUserInSession(c, user); err != nil {
		c.HTML(http.StatusInternalServerError, "profile.html", gin.H{
			"error": "Profile updated but failed to update session.",
			"user":  user,
			"title": "Profile",
		})
		return
	}

	c.HTML(http.StatusOK, "profile.html", gin.H{
		"success": "Profile updated successfully",
		"title":   "Profile",
		"user":    user,
	})
}

func ChangePassword(c *gin.Context) {
	// Retrieve the user from the session
	user, err := utils.GetUserFromSession(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
		return
	}

	// Bind input
	var input struct {
		OldPassword string `form:"old_password" binding:"required"`
		NewPassword string `form:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "change_password.html", gin.H{
			"error": "Please ensure your new password is at least 6 characters long",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Fetch the latest user record from the database
	var freshUser models.User
	if err := config.DB.First(&freshUser, "id = ?", user.ID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "Unable to verify user credentials. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}
	user = &freshUser

	// Validate the old password
	if err := user.CheckPassword(input.OldPassword); err != nil {
		c.HTML(http.StatusUnauthorized, "change_password.html", gin.H{
			"error": "The current password you entered is incorrect",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Check if the new password is different from the old password
	if input.OldPassword == input.NewPassword {
		c.HTML(http.StatusBadRequest, "change_password.html", gin.H{
			"error": "Your new password must be different from your current password",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Fetch the last 5 password histories
	var passwordHistories []models.PasswordHistory
	if err := config.DB.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		// Limit(5).
		Find(&passwordHistories).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "Unable to verify password history. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Compare the plain new password with previous passwords using the model function
    if user.IsPasswordUsed(input.NewPassword, passwordHistories) {
        c.HTML(http.StatusBadRequest, "change_password.html", gin.H{
            "error": "This password has been used recently. Please choose a different password",
            "title": "Change Password",
            "user": user,
        })
        return
    }

	// Begin a transaction
	tx := config.DB.Begin()

	// Update the user's password
	user.Password = input.NewPassword
	if err := user.HashPassword(); err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "An error occurred while securing your password. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "Unable to update your password. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Add the old password to the password history
	newHistory := models.PasswordHistory{
		UserID:   user.ID,
		Password: user.Password,
	}
	if err := tx.Create(&newHistory).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "Unable to update your password. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "change_password.html", gin.H{
			"error": "Unable to complete the password update. Please try again",
			"title": "Change Password",
			"user":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "change_password.html", gin.H{
		"success": "Your password has been successfully updated",
		"title":   "Change Password",
		"user":    user,
	})
}