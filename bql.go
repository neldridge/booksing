package main

import (
	"strings"

	"github.com/globalsign/mgo/bson"
)

func parseQuery(s string) bson.M {
	q := bson.M{}
	params := strings.Split(s, ",")
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(parts[0])
		filter := strings.TrimSpace(parts[1])

		q[field] = bson.M{
			"$regex":   filter,
			"$options": "i",
		}
	}

	return q
}
