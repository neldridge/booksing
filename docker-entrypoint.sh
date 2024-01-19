#!/bin/sh

# Create directories based on environment variables
mkdir -p -m 1755 ${BOOKSING_BOOKDIR} ${BOOKSING_DATABASEDIR} ${BOOKSING_FAILDIR} ${BOOKSING_IMPORTDIR}

# Run the Golang app
exec "$@"
