package handler

import (
	"context"
	"strconv"

	"gonio/internal/model"
	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/i18n"
	"gonio/internal/pkg/logger"
	"gonio/internal/pkg/req"
	"gonio/internal/pkg/response"
	"gonio/internal/pkg/validator"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type productService interface {
	List(ctx context.Context, page, size int) ([]model.Product, int64, error)
	GetByID(ctx context.Context, id uint) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uint) error
}

type ProductHandler struct {
	productSvc productService
}

func NewProductHandler(productSvc productService) *ProductHandler {
	return &ProductHandler{productSvc: productSvc}
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
		writeServiceError(c, err)
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
	logger.WithCtx(c.Request.Context()).Infow("product created",
		zap.Uint("product_id", product.ID),
		zap.String("product_name", product.Name),
	)
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
		writeServiceError(c, err)
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
		writeServiceError(c, err)
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
		writeServiceError(c, err)
		return
	}
	response.Success(c, nil)
}
