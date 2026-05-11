package database

import (
	"context"

	"gorm.io/gorm"
)

// txKey 用于在 context 中存储事务 DB 实例的 key
type txKey struct{}

// TxFunc 事务执行函数类型
type TxFunc func(ctx context.Context) error

// WithTransaction 在事务中执行函数，自动处理提交和回滚。
// 使用方式：
//
//	err := database.WithTransaction(ctx, db, func(txCtx context.Context) error {
//	    // 在 txCtx 中执行多个 Repository 操作
//	    if err := repo1.Create(txCtx, data1); err != nil {
//	        return err // 自动回滚
//	    }
//	    if err := repo2.Update(txCtx, data2); err != nil {
//	        return err // 自动回滚
//	    }
//	    return nil // 自动提交
//	})
func WithTransaction(ctx context.Context, db *gorm.DB, fn TxFunc) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务 DB 注入到 context
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// GetDB 从 context 获取 DB 实例。
// 如果 context 中存在事务 DB，则返回事务 DB；否则返回普通 DB。
// Repository 层应该使用此方法获取 DB 实例，以支持事务和非事务场景。
//
// 使用方式：
//
//	func (r *productRepo) Create(ctx context.Context, product *model.Product) error {
//	    db := database.GetDB(ctx, r.db)
//	    return db.Create(product).Error
//	}
func GetDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
