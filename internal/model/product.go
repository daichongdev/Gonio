package model

// Product 商品模型
type Product struct {
	BaseModel
	Name        string  `json:"name" gorm:"size:255;not null;index:idx_name;comment:商品名称"`
	Description string  `json:"description" gorm:"type:text;comment:商品描述"`
	Price       float64 `json:"price" gorm:"type:decimal(10,2);not null;index:idx_price;comment:价格"`
	Stock       int     `json:"stock" gorm:"default:0;comment:库存"`
	Status      int8    `json:"status" gorm:"default:1;index:idx_category_status;comment:状态 1-上架 0-下架"`
	CategoryID  uint    `json:"category_id" gorm:"index:idx_category_status;comment:分类ID"`
}

func (Product) TableName() string {
	return "products"
}
