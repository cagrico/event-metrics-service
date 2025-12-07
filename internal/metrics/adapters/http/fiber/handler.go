package fiber

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"event-metrics-service/internal/metrics/core/domain"
	"event-metrics-service/internal/metrics/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type GetMetricsUseCase interface {
	Execute(ctx context.Context, in usecase.GetMetricsInput) (*domain.AggregatedMetrics, error)
}

type MetricsHandler struct {
	uc GetMetricsUseCase
}

func NewMetricsHandler(uc GetMetricsUseCase) *MetricsHandler {
	return &MetricsHandler{uc: uc}
}

// GetMetrics godoc
// @Summary Query aggregated metrics
// @Description Returns metrics grouped by channel or time bucket
// @Tags Metrics
// @Accept json
// @Produce json
// @Param event_name query string true "Event name"
// @Param from query int true "From timestamp"
// @Param to query int true "To timestamp"
// @Param group_by query string false "Group by: channel | time"
// @Param interval query string false "Interval: minute | hour | day"
// @Success 200 {object} MetricsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /metrics [get]
func (h *MetricsHandler) GetMetrics(c *fiber.Ctx) error {
	eventName := c.Query("event_name", "")
	if eventName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "event_name is required",
		})
	}

	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")
	if fromStr == "" || toStr == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "from and to are required",
		})
	}

	from, err := strconv.ParseInt(fromStr, 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid 'from' parameter",
		})
	}
	to, err := strconv.ParseInt(toStr, 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid 'to' parameter",
		})
	}

	var channelPtr *string
	channel := c.Query("channel", "")
	if channel != "" {
		channelPtr = &channel
	}

	groupBy := c.Query("group_by", "")
	interval := c.Query("interval", "")

	in := usecase.GetMetricsInput{
		EventName: eventName,
		From:      from,
		To:        to,
		Channel:   channelPtr,
		GroupBy:   groupBy,
		Interval:  interval,
	}

	res, err := h.uc.Execute(c.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidMetricsQuery),
			errors.Is(err, usecase.ErrInvalidTimeRange),
			errors.Is(err, usecase.ErrInvalidGroupBy),
			errors.Is(err, usecase.ErrInvalidInterval):
			return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
				Error:   "invalid_event",
				Message: err.Error(),
			})
		default:
			return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
				Error: "internal_server_error",
			})
		}
	}

	resp := MetricsResponse{
		EventName:   res.EventName,
		From:        res.From,
		To:          res.To,
		TotalCount:  res.TotalCount,
		UniqueUsers: res.UniqueUsers,
		GroupBy:     res.GroupBy,
		Groups:      make([]MetricsGroupResponse, 0, len(res.Groups)),
	}

	for _, g := range res.Groups {
		resp.Groups = append(resp.Groups, MetricsGroupResponse{
			Key:         g.Key,
			TotalCount:  g.TotalCount,
			UniqueUsers: g.UniqueUsers,
		})
	}

	return c.Status(http.StatusOK).JSON(resp)
}
