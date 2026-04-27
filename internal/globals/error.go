package globals

import "errors"

var (
	// ErrNotFound 记录不存在
	ErrNotFound = errors.New("record not found")

	// ErrEntityNil 实体为空
	ErrEntityNil = errors.New("entity cannot be nil")

	// ErrInvalidPageSize 无效的分页大小
	ErrInvalidPageSize = errors.New("invalid page size")

	// ErrInvalidPage 无效的页码
	ErrInvalidPage = errors.New("invalid page number")

	// ErrTransactionFailed 事务失败
	ErrTransactionFailed = errors.New("transaction failed")

	ErrInvalidKey = errors.New("id cannot be nil")
)

// IsNotFound 检查是否是记录不存在错误
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
