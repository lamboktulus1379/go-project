package middleware

import (
	"errors"
	"fmt"
	"log"
	"my-project/domain/dto"
	"my-project/domain/model"
	"my-project/domain/repository"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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
		var userClaims model.UserClaims
		token, err := jwt.ParseWithClaims(auth[1], &userClaims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		fmt.Printf("Claims: %+v\n", userClaims)

		if token.Valid {
			fmt.Println("You look nice today")
			_, err := userRepository.GetByUserName(ctx.Request.Context(), userClaims.UserName)
			if err != nil {
				fmt.Println("User not found")
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
			ctx.Set("user_id", userClaims.Issuer)
			ctx.Next()
		} else {
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
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
		}

	}
}
