package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	booksProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "booksing_books_processed",
		Help: "The number of processed books",
	}, []string{"transaction"})
	booksProcessedTime = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "booksing_books_duration_seconds",
		Help: "The time taken to process the books in seconds",
	}, []string{"transaction"})
	meiliErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "booksing_meili_errors",
		Help: "The number of errors encountered when contacting meilisearch",
	}, []string{"type"})
	dbErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "booksing_db_errors",
		Help: "The number of errors encountered when using the db",
	}, []string{"type"})
)
