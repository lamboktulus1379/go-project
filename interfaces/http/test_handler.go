package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"my-project/usecase"
)

type ITestHandler interface {
	Test(c *gin.Context)
}

type TestHandler struct {
	TestUsecase usecase.ITestUsecase
}

func NewTestHandler(testUsecase usecase.ITestUsecase) ITestHandler {
	return &TestHandler{TestUsecase: testUsecase}
}

func (testHandler *TestHandler) Test(c *gin.Context) {
	res := testHandler.TestUsecase.Test(c.Request.Context())
	c.JSON(http.StatusOK, res)
}
