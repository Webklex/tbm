package app

import (
	"fmt"
	"sort"
	"strconv"
)

type Paginator struct {
	Path       string
	Parameters map[string]string
	Limit      int
	Page       int
	Total      int
	TotalPages int
	Offset     int
	data       []interface{}
}

type PaginatorLinks struct {
	Page     int
	Url      string
	Label    string
	Disabled bool
}

const (
	DefaultPaginatorLength int = 8
)

func NewPaginator(limit, page int) *Paginator {
	if limit <= 0 {
		limit = DefaultPaginatorLength
	}
	if page < 0 {
		page = 0
	}
	return &Paginator{
		Path:       "",
		Parameters: map[string]string{},
		Limit:      limit,
		Page:       page,
		Total:      0,
		TotalPages: 0,
		Offset:     0,
		data:       make([]interface{}, 0),
	}
}

func (p *Paginator) SetData(data []interface{}) {
	p.Offset = p.Limit * p.Page
	p.Total = len(data)
	p.TotalPages = p.Total / p.Limit
	p.data = data
}

func (p *Paginator) Sort(less func(i, j interface{}) bool) {
	sort.Slice(p.data, func(i, j int) bool {
		return less(p.data[i], p.data[j])
	})
}

func (p *Paginator) Data() []interface{} {
	if p.TotalPages < p.Page || p.Total <= p.Offset {
		return []interface{}{}
	}
	data := p.data[p.Offset:]
	if len(data) > p.Limit {
		data = data[0:p.Limit]
	}
	return data
}

func (p *Paginator) GetParameter(name string) string {
	if s, ok := p.Parameters[name]; ok {
		return s
	}
	return ""
}

func (p *Paginator) Links(numberOfLinks int) []*PaginatorLinks {
	links := make([]*PaginatorLinks, 0)
	parameters := ""
	for k, s := range p.Parameters {
		if parameters == "" {
			parameters = k + "=" + s
		} else {
			parameters += "&" + k + "=" + s
		}
	}

	/**
	 * Set the first button / backwards 1
	 */
	if p.Page == 0 {
		links = append(links, &PaginatorLinks{
			Page:     0,
			Url:      fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, 0, p.Limit, parameters),
			Label:    "<span class='fa fa-chevron-left'></span>",
			Disabled: true,
		})
	} else {
		links = append(links, &PaginatorLinks{
			Page:  p.Page - 1,
			Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, p.Page-1, p.Limit, parameters),
			Label: "<span class='fa fa-chevron-left'></span>",
		})
	}

	deltaStart := p.Page - numberOfLinks
	deltaEnd := p.Page + numberOfLinks

	if deltaStart < 0 {
		// No left ... button needed
		deltaEnd += deltaStart + numberOfLinks

		if deltaStart >= numberOfLinks*-1 {
			// Draw left buttons up til the current page
			deltaStart = 0
		}
	} else {
		links = append(links, &PaginatorLinks{
			Page:  0,
			Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, 0, p.Limit, parameters),
			Label: "1",
		})
		links = append(links, &PaginatorLinks{
			Page:     0,
			Url:      "",
			Label:    "...",
			Disabled: true,
		})
	}

	for i := deltaStart; i < p.Page; i++ {
		links = append(links, &PaginatorLinks{
			Page:  i,
			Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, i, p.Limit, parameters),
			Label: strconv.Itoa(i + 1),
		})
	}

	// Draw the current page link
	links = append(links, &PaginatorLinks{
		Page:  p.Page,
		Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, p.Page, p.Limit, parameters),
		Label: strconv.Itoa(p.Page + 1),
	})

	if deltaEnd >= p.TotalPages-1 {
		// No right ... button needed
		deltaEnd = p.TotalPages - 1
	}

	for i := p.Page + 1; i <= deltaEnd; i++ {
		links = append(links, &PaginatorLinks{
			Page:  i,
			Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, i, p.Limit, parameters),
			Label: strconv.Itoa(i + 1),
		})
	}

	if deltaEnd < p.TotalPages-1 {
		links = append(links, &PaginatorLinks{
			Page:     0,
			Url:      "",
			Label:    "...",
			Disabled: true,
		})
	}

	if p.Page == p.TotalPages {
		links = append(links, &PaginatorLinks{
			Page:     p.TotalPages,
			Url:      fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, p.TotalPages, p.Limit, parameters),
			Label:    "<span class='fa fa-chevron-right'></span>",
			Disabled: true,
		})
	} else {
		if deltaEnd < p.TotalPages {
			links = append(links, &PaginatorLinks{
				Page:  p.TotalPages,
				Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, p.TotalPages, p.Limit, parameters),
				Label: strconv.Itoa(p.TotalPages + 1),
			})
		}
		links = append(links, &PaginatorLinks{
			Page:  p.Page + 1,
			Url:   fmt.Sprintf("%s?page=%d&limit=%d&%s", p.Path, p.Page+1, p.Limit, parameters),
			Label: "<span class='fa fa-chevron-right'></span>",
		})
	}

	return links
}
