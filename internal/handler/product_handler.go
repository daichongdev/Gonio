package handler

import (
	"context"
	"errors"
	"strconv"

	"goflow/internal/model"
	"goflow/internal/mq"
	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/i18n"
	"goflow/internal/pkg/logger"
	"goflow/internal/pkg/req"
	"goflow/internal/pkg/response"
	"goflow/internal/pkg/validator"

	"github.com/gin-gonic/gin"
)

type productService interface {
	List(ctx context.Context, page, size int) ([]model.Product, int64, error)
	GetByID(ctx context.Context, id uint) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uint) error
}

type productTaskPublisher interface {
	PublishEmail(ctx context.Context, payload mq.EmailPayload) error
}

type ProductHandler struct {
	productSvc  productService
	mqPublisher productTaskPublisher
}

func NewProductHandler(productSvc productService, mqPublisher productTaskPublisher) *ProductHandler {
	return &ProductHandler{productSvc: productSvc, mqPublisher: mqPublisher}
}

// List 商品列表
func (h *ProductHandler) List(c *gin.Context) {
	page, size := req.ParsePage(c)

	products, total, err := h.productSvc.List(c.Request.Context(), page, size)
	if err != nil {
		response.Error(c, errcode.ErrInternal())
		return
	}
	response.SuccessWithPage(c, products, total, page, size)
}

// Get 商品详情
func (h *ProductHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrBadRequest())
		return
	}

	product, err := h.productSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		var appErr *errcode.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errcode.ErrInternal())
		return
	}
	response.Success(c, product)
}

// Create 创建商品
func (h *ProductHandler) Create(c *gin.Context) {
	var request req.CreateReq
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ErrorWithMsg(c, 400, errcode.CodeBadRequest, validator.Translate(err, i18n.GetLang(c)))
		return
	}

	product := &model.Product{
		Name:        request.Name,
		Description: request.Description,
		Price:       request.Price,
		Stock:       request.Stock,
		CategoryID:  request.CategoryID,
		Status:      1,
	}
	if err := h.productSvc.Create(c.Request.Context(), product); err != nil {
		response.Error(c, errcode.ErrInternal())
		return
	}
	if h.mqPublisher != nil {
		if err := h.mqPublisher.PublishEmail(c.Request.Context(), mq.EmailPayload{
			To:      "daichongweb@qq.com",
			Subject: "标题",
			Body:    product.Description,
		}); err != nil {
			logger.WithCtx(c.Request.Context()).Warnw("publish email task failed", "error", err)
		}
	}
	response.Success(c, product)
}

// Update 更新商品
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrBadRequest())
		return
	}

	product, err := h.productSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		var appErr *errcode.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errcode.ErrInternal())
		return
	}

	var request req.UpdateReq
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ErrorWithMsg(c, 400, errcode.CodeBadRequest, validator.Translate(err, i18n.GetLang(c)))
		return
	}

	if request.Name != nil {
		product.Name = *request.Name
	}
	if request.Description != nil {
		product.Description = *request.Description
	}
	if request.Price != nil {
		product.Price = *request.Price
	}
	if request.Stock != nil {
		product.Stock = *request.Stock
	}
	if request.Status != nil {
		product.Status = *request.Status
	}
	if request.CategoryID != nil {
		product.CategoryID = *request.CategoryID
	}

	if err := h.productSvc.Update(c.Request.Context(), product); err != nil {
		var appErr *errcode.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errcode.ErrInternal())
		return
	}
	response.Success(c, product)
}

// Delete 删除商品
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrBadRequest())
		return
	}

	if err := h.productSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		var appErr *errcode.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errcode.ErrInternal())
		return
	}
	response.Success(c, nil)
}
