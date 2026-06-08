package usecase

import (
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

type RegisterCmd struct {
	Name     string
	Email    string
	Password string
}

type LoginCmd struct {
	Email    string
	Password string
}

type LogoutCmd struct {
}

type CurrentUserQry struct {
	UserID string
}

type AuthStatusQry struct {
	UserID string
}

type ForgotPasswordCmd struct {
	Email string
}

type ResetPasswordCmd struct {
	Token    string
	Password string
}

type AuthCo struct {
	User UserCo
}

type AuthStatusCo struct {
	LoggedIn bool
	User     *UserCo
}

type ForgotPasswordCo struct {
	ResetToken string
	User       UserCo
}

func Register(ctx fwusecase.Context, cmd RegisterCmd) (AuthCo, error) {
	if cmd.Name == "" || cmd.Email == "" || cmd.Password == "" {
		return AuthCo{}, fwusecase.E(fwusecase.CodeValidation, "name, email, and password are required", nil)
	}
	if len(cmd.Password) < 6 {
		return AuthCo{}, fwusecase.E(fwusecase.CodeValidation, "password must be at least 6 characters", nil)
	}

	exists, err := models.UserExistsByEmail(ctx.Std(), cmd.Email)
	if err != nil {
		return AuthCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to check email", err)
	}
	if exists {
		return AuthCo{}, fwusecase.E(fwusecase.CodeValidation, "email is already registered", nil)
	}

	user := &models.User{
		Name:  cmd.Name,
		Email: cmd.Email,
	}
	if err := user.SetPassword(cmd.Password); err != nil {
		return AuthCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to process password", err)
	}
	if err := models.CreateUser(ctx.Std(), user); err != nil {
		return AuthCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to create user", err)
	}

	userCo, err := GetUser(ctx, UserDetailQry{ID: user.ID})
	if err != nil {
		return AuthCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load user", err)
	}

	return AuthCo{User: userCo}, nil
}

func Login(ctx fwusecase.Context, cmd LoginCmd) (AuthCo, error) {
	if cmd.Email == "" || cmd.Password == "" {
		return AuthCo{}, fwusecase.E(fwusecase.CodeValidation, "email and password are required", nil)
	}

	user, err := models.GetUserWithPasswordByEmail(ctx.Std(), cmd.Email)
	if err != nil {
		return AuthCo{}, fwusecase.E(fwusecase.CodeInternal, "login failed", err)
	}
	if user == nil || !user.CheckPassword(cmd.Password) {
		return AuthCo{}, fwusecase.E(fwusecase.CodeUnauthorized, "email or password is incorrect", nil)
	}
	if user.IsActive == 0 {
		return AuthCo{}, fwusecase.E(fwusecase.CodeForbidden, "account is disabled", nil)
	}

	return AuthCo{User: userCoFromModel(user)}, nil
}

func Logout(ctx fwusecase.Context, cmd LogoutCmd) error {
	return nil
}

func GetCurrentUser(ctx fwusecase.Context, qry CurrentUserQry) (UserCo, error) {
	if qry.UserID == "" {
		return UserCo{}, fwusecase.E(fwusecase.CodeUnauthorized, "not logged in", nil)
	}

	user, err := GetUser(ctx, UserDetailQry{ID: qry.UserID})
	if err != nil {
		if fwusecase.CodeOf(err) == fwusecase.CodeNotFound {
			return UserCo{}, fwusecase.E(fwusecase.CodeUnauthorized, "not logged in", err)
		}
		return UserCo{}, err
	}
	return user, nil
}

func AuthStatus(ctx fwusecase.Context, qry AuthStatusQry) (AuthStatusCo, error) {
	if qry.UserID == "" {
		return AuthStatusCo{LoggedIn: false}, nil
	}

	user, err := GetUser(ctx, UserDetailQry{ID: qry.UserID})
	if err != nil {
		if fwusecase.CodeOf(err) == fwusecase.CodeNotFound || fwusecase.CodeOf(err) == fwusecase.CodeUnauthorized {
			return AuthStatusCo{LoggedIn: false}, nil
		}
		return AuthStatusCo{}, err
	}
	return AuthStatusCo{
		LoggedIn: true,
		User:     &user,
	}, nil
}

func ForgotPassword(ctx fwusecase.Context, cmd ForgotPasswordCmd) (ForgotPasswordCo, error) {
	if cmd.Email == "" {
		return ForgotPasswordCo{}, fwusecase.E(fwusecase.CodeValidation, "email is required", nil)
	}

	user, err := models.GetUserWithPasswordByEmail(ctx.Std(), cmd.Email)
	if err != nil || user == nil {
		return ForgotPasswordCo{}, nil
	}

	token, err := models.CreatePasswordReset(ctx.Std(), user.ID)
	if err != nil {
		return ForgotPasswordCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to create reset link", err)
	}

	return ForgotPasswordCo{
		ResetToken: token,
		User:       userCoFromModel(user),
	}, nil
}

func ResetPassword(ctx fwusecase.Context, cmd ResetPasswordCmd) error {
	if cmd.Token == "" || cmd.Password == "" {
		return fwusecase.E(fwusecase.CodeValidation, "token and password are required", nil)
	}
	if len(cmd.Password) < 6 {
		return fwusecase.E(fwusecase.CodeValidation, "password must be at least 6 characters", nil)
	}

	reset, err := models.VerifyPasswordResetToken(ctx.Std(), cmd.Token)
	if err != nil {
		return fwusecase.E(fwusecase.CodeValidation, "reset link is invalid or expired", err)
	}

	if err := models.UpdateUserPassword(ctx.Std(), reset.UserID, cmd.Password); err != nil {
		return fwusecase.E(fwusecase.CodeInternal, "failed to update password", err)
	}

	_ = models.MarkPasswordResetUsed(ctx.Std(), reset.ID)
	return nil
}
