package pagination

const DefaultPaginatorLimit = 10

var userDefinePaginatorLimit int64

type AbstractPaginator interface {
	GetLimit() int64
	GetTotal() int64
	GetOffset() int64
	GetCurrentPage() int64
}

type Paginator struct {
	Page   int64 `query:"page" json:"page"`
	Last   int64 `query:"-" json:"last"`
	Total  int64 `query:"-" json:"total"`
	Limit  int64 `query:"limit" json:"limit"`
	Offset int64 `query:"-" json:"-"`
}

func Init() *Paginator {
	return &Paginator{}
}

func SetUserDefineLimit(limit int64) {
	userDefinePaginatorLimit = limit
}

func (receiver *Paginator) GetLimit() int64 {
	if receiver.Limit == 0 {
		limit := userDefinePaginatorLimit
		if limit == 0 {
			limit = DefaultPaginatorLimit
		}

		receiver.Limit = limit
	}

	return receiver.Limit
}

func (receiver *Paginator) GetTotal() int64 {
	return receiver.Total
}

func (receiver *Paginator) GetOffset() int64 {
	return (receiver.GetCurrentPage() - 1) * receiver.Limit
}

func (receiver *Paginator) GetCurrentPage() int64 {
	if receiver.Page == 0 {
		receiver.Page = 1
	}

	return receiver.Page
}
