package middlewares

import (
	"fmt"
	"github.com/Xhofe/alist/model"
	"github.com/Xhofe/alist/server/common"
	"github.com/gin-gonic/gin"
)

func CheckAccount(c *gin.Context) {
	if model.AccountsCount() == 0 {
		common.ErrorResp(c, fmt.Errorf("no accounts,please add one first"), 1001)
		return
	}
	c.Next()
}