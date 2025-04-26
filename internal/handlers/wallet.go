package handlers

import (
	"Crypto.com/internal/repositories/postgres"
	"errors"
	"net/http"
	"strconv"

	"Crypto.com/internal/services"
	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	service *services.WalletService
}

func NewWalletHandler(service *services.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

func (h *WalletHandler) Deposit(c *gin.Context) {
	userID := c.Param("userID")

	var request struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Deposit(c.Request.Context(), userID, request.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *WalletHandler) Withdraw(c *gin.Context) {
	userID := c.Param("userID")

	var request struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Withdraw(c.Request.Context(), userID, request.Amount); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "insufficient funds" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	senderID := c.Param("userID")

	var request struct {
		ReceiverID string  `json:"receiver_id" binding:"required"`
		Amount     float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Transfer(c.Request.Context(), senderID, request.ReceiverID, request.Amount); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "insufficient funds for transfer" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID := c.Param("userID")

	balance, err := h.service.GetBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *WalletHandler) TransactionHistory(c *gin.Context) {
	userID := c.Param("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	transactions, err := h.service.GetTransactionHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		// Handle specific error cases
		if errors.Is(err, postgres.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"page":         page,
		"limit":        limit,
		"total":        len(transactions),
	})
}
