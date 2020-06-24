# booksing
A tool to browse epubs and convert them to mobi ebooks using kindlegen. 

Heavily inspired by https://github.com/geek1011/BookBrowser/

## installation
Download an appropriate release from the [release](https://github.com/gnur/booksing/releases) page


## Features
- Easy-to-use
- Searches for query in authors name and title of book
- List view
- "Responsive" web interface
- automatic deletion of duplicates and unparsable epubs

## Requirements
Booksing only depends on [meilisearch](https://www.meilisearch.com/).

## configuration

Set the following env vars to configure booksing:

- BOOKSING_LOGLEVEL (error, warning, info, debug)
- BOOKSING_ADMINUSER (this email address can admin other users)
- BOOKSING_DATABASE (path to store the database file (used to store users & downloads))
- BOOKSING_IMPORTDIR (path where booksing should load new books from)
- BOOKSING_BOOKDIR (path where booksing can move the files after loading)
- BOOKSING_MEILI_HOST (meilisearch hostname, must include protocol, usually something like http://localhost:7700)
- BOOKSING_MEILI_INDEX (index to store books in)
- BOOKSING_MEILI_KEY (access key for meilisearch, must have write access)
- BOOKSING_BINDADDRESS (address to bind on, default: `localhost:7132`)
- BOOKSING_TIMEZONE (timezone, default: `Europe/Amsterdam`)

## Usage
1. Run the booksing binary from the directory with the epub books. You can access the web interface at [http://localhost:7132](http://localhost:7132) 
