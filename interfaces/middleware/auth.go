package middleware

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"my-project/domain/dto"
	"my-project/domain/model"
	"my-project/domain/repository"
)

func Auth(userRepository repository.IUser) gin.HandlerFunc {
	var res dto.Res
	res.ResponseCode = "401"
	res.ResponseMessage = "Unauthorized"

	log.Println("Inside auth middleware")
	return func(ctx *gin.Context) {
		authorization := ctx.Request.Header.Get("Authorization")
		secretKey := os.Getenv("SECRET_KEY")
		if authorization == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
			return
		}
		auth := strings.Split(authorization, "Bearer ")
		fmt.Println("Auth:", auth[1])
		if len(auth) != 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
			return
		}
		userClaims, token, err := getClaim(auth, secretKey)

		if token.Valid {
			if !next(ctx, userRepository, userClaims) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
		} else {
			if abort(err, res) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
		}
	}
}

func abort(err error, res dto.Res) bool {
	var ve *jwt.ValidationError
	if errors.As(err, &ve) {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			res.ResponseMessage = "That's not even a token"
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			res.ResponseMessage = "Timing is everything"
		} else {
			res.ResponseMessage = fmt.Sprintf("Couldn't handle this token:%v", err)
		}
		return true
	}
	return false
}

func next(ctx *gin.Context, userRepository repository.IUser, userClaims model.UserClaims) bool {
	fmt.Println("You look nice today")
	_, err := userRepository.GetByUserName(ctx.Request.Context(), userClaims.UserName)
	if err != nil {
		fmt.Println("User not found")
		return true
	}
	ctx.Set("user_id", userClaims.Issuer)
	ctx.Next()
	return false
}

func getClaim(auth []string, secretKey string) (model.UserClaims, *jwt.Token, error) {
	var userClaims model.UserClaims
	token, err := jwt.ParseWithClaims(
		auth[1],
		&userClaims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	fmt.Printf("Claims: %+v\n", userClaims)
	return userClaims, token, err
}
