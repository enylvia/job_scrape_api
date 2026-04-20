package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"job_aggregator/internal/models"
	"job_aggregator/internal/repository"
)

type AboutHandler struct {
	logger    *log.Logger
	aboutRepo *repository.AboutPageRepository
}

type aboutPageRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewAboutHandler(logger *log.Logger, aboutRepo *repository.AboutPageRepository) *AboutHandler {
	return &AboutHandler{
		logger:    logger,
		aboutRepo: aboutRepo,
	}
}

func (h *AboutHandler) List(w http.ResponseWriter, r *http.Request) {
	pages, totalCount, err := h.aboutRepo.List(r.Context())
	if err != nil {
		h.logger.Printf("about handler: list pages error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to list about pages")
		return
	}

	writeData(w, http.StatusOK, "about pages fetched successfully", totalCount, pages)
}

func (h *AboutHandler) Get(w http.ResponseWriter, r *http.Request) {
	pageID, err := parseAboutPageID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := h.aboutRepo.GetByID(r.Context(), pageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "about page not found")
			return
		}

		h.logger.Printf("about handler: get page id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to get about page")
		return
	}

	writeData(w, http.StatusOK, "about page fetched successfully", 1, page)
}

func (h *AboutHandler) Create(w http.ResponseWriter, r *http.Request) {
	page, err := decodeAboutPageRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pageID, err := h.aboutRepo.Create(r.Context(), page)
	if err != nil {
		h.logger.Printf("about handler: create page error=%v", err)
		writeError(w, http.StatusInternalServerError, "failed to create about page")
		return
	}

	createdPage, err := h.aboutRepo.GetByID(r.Context(), pageID)
	if err != nil {
		h.logger.Printf("about handler: reload created page id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to reload about page")
		return
	}

	writeData(w, http.StatusCreated, "about page created successfully", 1, createdPage)
}

func (h *AboutHandler) Update(w http.ResponseWriter, r *http.Request) {
	pageID, err := parseAboutPageID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.aboutRepo.GetByID(r.Context(), pageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "about page not found")
			return
		}

		h.logger.Printf("about handler: load page for update id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to load about page")
		return
	}

	page, err := decodeAboutPageRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page.ID = pageID

	if err := h.aboutRepo.Update(r.Context(), page); err != nil {
		h.logger.Printf("about handler: update page id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to update about page")
		return
	}

	updatedPage, err := h.aboutRepo.GetByID(r.Context(), pageID)
	if err != nil {
		h.logger.Printf("about handler: reload updated page id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to reload about page")
		return
	}

	writeData(w, http.StatusOK, "about page updated successfully", 1, updatedPage)
}

func (h *AboutHandler) Delete(w http.ResponseWriter, r *http.Request) {
	pageID, err := parseAboutPageID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := h.aboutRepo.GetByID(r.Context(), pageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "about page not found")
			return
		}

		h.logger.Printf("about handler: load page for delete id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to load about page")
		return
	}

	if err := h.aboutRepo.Delete(r.Context(), pageID); err != nil {
		h.logger.Printf("about handler: delete page id=%d error=%v", pageID, err)
		writeError(w, http.StatusInternalServerError, "failed to delete about page")
		return
	}

	writeData(w, http.StatusOK, "about page deleted successfully", 1, page)
}

func decodeAboutPageRequest(r *http.Request) (models.AboutPage, error) {
	var request aboutPageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.AboutPage{}, fmt.Errorf("invalid request body")
	}

	title := strings.TrimSpace(request.Title)
	if title == "" {
		return models.AboutPage{}, fmt.Errorf("title is required")
	}

	return models.AboutPage{
		Title: title,
		Body:  strings.TrimSpace(request.Body),
	}, nil
}

func parseAboutPageID(r *http.Request) (int64, error) {
	rawID := strings.TrimSpace(r.PathValue("id"))
	if rawID == "" {
		return 0, fmt.Errorf("about page id is required")
	}

	pageID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || pageID <= 0 {
		return 0, fmt.Errorf("about page id must be a positive integer")
	}

	return pageID, nil
}
