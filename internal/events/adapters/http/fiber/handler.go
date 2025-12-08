package fiber

import (
	"context"
	"errors"
	"net/http"

	"event-metrics-service/internal/events/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type StoreEventUseCase interface {
	Execute(ctx context.Context, in usecase.StoreEventInput) (bool, error)
	BulkCreateEvents(ctx context.Context, in usecase.BulkCreateEventsInput) (usecase.BulkCreateEventsResult, error)
}

type EventHandler struct {
	storeUC StoreEventUseCase
}

func NewEventHandler(storeUC StoreEventUseCase) *EventHandler {
	return &EventHandler{storeUC: storeUC}
}

// CreateEvent godoc
// @Summary Create a new event
// @Description Stores a single event with idempotency handling
// @Tags Events
// @Accept json
// @Produce json
// @Param request body CreateEventRequest true "Event payload"
// @Success 201 {object} CreateEventResponse
// @Success 200 {object} CreateEventResponse "Duplicate event"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /events [post]
func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	var req CreateEventRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_json",
		})
	}

	input := usecase.StoreEventInput{
		EventName:  req.EventName,
		Channel:    req.Channel,
		CampaignID: req.CampaignID,
		UserID:     req.UserID,
		Timestamp:  req.Timestamp,
		Tags:       req.Tags,
		Metadata:   req.Metadata,
	}

	created, err := h.storeUC.Execute(c.UserContext(), input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidEvent),
			errors.Is(err, usecase.ErrFutureTime):
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

	if !created {
		resp := CreateEventResponse{
			Status: "duplicate",
		}
		return c.Status(http.StatusOK).JSON(resp)
	}

	resp := CreateEventResponse{
		Status: "created",
	}
	return c.Status(http.StatusCreated).JSON(resp)
}

// BulkCreateEvents godoc
// @Summary Bulk create events
// @Description Accepts a list of events and stores them individually
// @Tags Events
// @Accept json
// @Produce json
// @Param request body BulkCreateEventsRequest true "Bulk event payload"
// @Success 201 {object} map[string]int
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /events/bulk [post]
func (h *EventHandler) BulkCreateEvents(c *fiber.Ctx) error {
	var req BulkCreateEventsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_json",
		})
	}

	if len(req.Events) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "events_list_required",
		})
	}

	inputs := make([]usecase.StoreEventInput, len(req.Events))
	for i, e := range req.Events {
		inputs[i] = usecase.StoreEventInput{
			EventName:  e.EventName,
			Channel:    e.Channel,
			CampaignID: e.CampaignID,
			UserID:     e.UserID,
			Timestamp:  e.Timestamp,
			Tags:       e.Tags,
			Metadata:   e.Metadata,
		}
	}

	result, err := h.storeUC.BulkCreateEvents(
		c.UserContext(),
		usecase.BulkCreateEventsInput{Events: inputs},
	)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidEvent),
			errors.Is(err, usecase.ErrFutureTime):
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"created":    result.Created,
		"duplicates": result.Duplicates,
	})
}
