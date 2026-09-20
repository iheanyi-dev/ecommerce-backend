package dto

// ListStoresInput contains the parameters used to search and paginate
// publicly visible stores.
//
// Query is an optional case-insensitive partial match against the Store name.
// Page starts at 1. PageSize controls how many stores are returned per page.
type ListStoresInput struct {
	Query    string
	Page     int
	PageSize int
}

// ListStoresOutput contains one page of publicly visible stores together
// with the pagination metadata required by the presentation layer.
type ListStoresOutput struct {
	Stores     []StoreOutput
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}
