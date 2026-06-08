package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	"github.com/tfnick/go-svelte-starter/api/framework/http/middleware"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	"github.com/tfnick/go-svelte-starter/api/framework/realtime"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type PointsResponse struct {
	UserID  string `json:"user_id"`
	Balance int64  `json:"balance"`
}

func ToPointsResponse(points usecase.PointsCo) PointsResponse {
	return PointsResponse{
		UserID:  points.UserID,
		Balance: points.Balance,
	}
}

func GetMyPoints(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	if currentUser == nil {
		return httpresponse.Unauthorized(c, "not logged in")
	}

	ctx := fwcontext.InternalUsecaseContext(c)
	points, err := usecase.GetUserPoints(ctx, usecase.PointsBalanceQry{UserID: currentUser.ID})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, ToPointsResponse(points))
}

func PointsSSE(c echo.Context) error {
	currentUser := middleware.GetCurrentUser(c)
	if currentUser == nil {
		return httpresponse.Unauthorized(c, "not logged in")
	}

	ctx := fwcontext.InternalUsecaseContext(c)
	initialPoints, err := usecase.GetUserPoints(ctx, usecase.PointsBalanceQry{UserID: currentUser.ID})
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}

	clientID := strings.TrimSpace(c.QueryParam("client_id"))
	if clientID == "" {
		clientID = uuid.Must(uuid.NewV7()).String()
	}

	return streamPointsSSE(c, currentUser.ID, clientID, initialPoints)
}

func streamPointsSSE(c echo.Context, userID string, clientID string, initialPoints usecase.PointsCo) error {
	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := res.Writer.(http.Flusher)
	if !ok {
		return httpresponse.InternalServerError(c, fmt.Errorf("response writer does not support streaming"), "streaming unsupported")
	}

	sub := realtime.SubscribeClient(userID, clientID)
	defer sub.Close()

	initialMessage, err := json.Marshal(realtime.NewPointsMessage(realtime.PointsPayload{
		UserID:   initialPoints.UserID,
		ClientID: sub.ClientID(),
		Balance:  initialPoints.Balance,
	}, ""))
	if err != nil {
		return httpresponse.InternalServerError(c, err, "failed to create points event")
	}

	res.WriteHeader(http.StatusOK)
	if err := writeSSEData(res, initialMessage); err != nil {
		return nil
	}
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case message, ok := <-sub.Messages:
			if !ok {
				return nil
			}
			if err := writeSSEData(res, message); err != nil {
				return nil
			}
			flusher.Flush()
		case <-heartbeat.C:
			if err := writeSSEComment(res, "keepalive"); err != nil {
				return nil
			}
			flusher.Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
}

func writeSSEData(w io.Writer, payload []byte) error {
	_, err := fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

func writeSSEComment(w io.Writer, comment string) error {
	_, err := fmt.Fprintf(w, ": %s\n\n", comment)
	return err
}
