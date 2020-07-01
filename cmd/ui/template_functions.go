package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"time"

	"github.com/gnur/booksing"
)

var templateFunctions = template.FuncMap{
	"percent": func(a, b int) float64 {
		return float64(a) / float64(b) * 100
	},
	"safeHTML": func(s interface{}) template.HTML {
		return template.HTML(fmt.Sprint(s))
	},
	"Iterate": func(offset, limit int64, results int) [][2]int64 {
		var i int64
		var Items [][2]int64
		if int64(results) == limit {
			offset += limit
		}
		for i = 0; i <= (offset / limit); i++ {
			Items = append(Items, [2]int64{
				i + 1,
				i * limit,
			})
		}
		if len(Items) > 8 {
			l := len(Items)
			return [][2]int64{
				Items[0],
				Items[1],
				Items[2],
				{-1, -1},
				Items[l-3],
				Items[l-2],
				Items[l-1],
			}
		}

		return Items
	},
	"prettyTime": func(s interface{}) template.HTML {
		t, ok := s.(time.Time)
		if !ok {
			return ""
		}
		if t.IsZero() {
			return template.HTML("never")
		}
		return template.HTML(t.Format("2006-01-02 15:04:05"))
	},
	"page": func(dir, q string, offset, limit int64) template.URL {
		v := url.Values{}
		v.Add("q", q)
		v.Add("l", fmt.Sprintf("%v", limit))
		if dir == "next" {
			start := offset + limit
			v.Add("o", fmt.Sprintf("%v", start))
		} else {
			start := offset - limit
			if start > 0 {
				v.Add("o", fmt.Sprintf("%v", start))
			}
		}
		return template.URL(v.Encode())

	},
	"json": func(s interface{}) template.HTML {
		json, _ := json.MarshalIndent(s, "", "  ")
		return template.HTML(string(json))
	},
	"icon": func(in *booksing.ShelveIcon) template.HTML {

		if in == nil {
			in = booksing.DefaultShelveIcon()
		}
		return template.HTML(fmt.Sprintf(`<svg class="bi %s" width="32" height="32" fill="currentColor">
                            <use xlink:href="/static/b-icons.svg#%s" />
                        </svg>`, in[1], in[0]))

	},
	"relativeTime": func(s interface{}) template.HTML {
		t, ok := s.(time.Time)
		if !ok {
			return ""
		}
		if t.IsZero() {
			return template.HTML("never")
		}
		tense := "ago"
		diff := time.Since(t)
		seconds := int64(diff.Seconds())
		if seconds < 0 {
			tense = "from now"
		}
		var quantifier string

		if seconds < 60 {
			quantifier = "s"
		} else if seconds < 3600 {
			quantifier = "m"
			seconds /= 60
		} else if seconds < 86400 {
			quantifier = "h"
			seconds /= 3600
		} else if seconds < 604800 {
			quantifier = "d"
			seconds /= 86400
		} else if seconds < 31556736 {
			quantifier = "w"
			seconds /= 604800
		} else {
			quantifier = "y"
			seconds /= 31556736
		}

		return template.HTML(fmt.Sprintf("%v%s %s", seconds, quantifier, tense))
	},
}
