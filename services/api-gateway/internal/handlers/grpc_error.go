package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trungquantrannguyen/threadly/pkg/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func HandleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Internal server error", err)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		response.Error(c, http.StatusBadRequest, st.Message(), err)

	case codes.Unauthenticated:
		response.Error(c, http.StatusUnauthorized, st.Message(), err)

	case codes.PermissionDenied:
		response.Error(c, http.StatusForbidden, st.Message(), err)

	case codes.NotFound:
		response.Error(c, http.StatusNotFound, st.Message(), err)

	case codes.AlreadyExists:
		response.Error(c, http.StatusConflict, st.Message(), err)

	default:
		response.Error(c, http.StatusInternalServerError, "Internal server error", err)
	}
}
